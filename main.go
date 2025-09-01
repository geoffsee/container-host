package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Ignition configuration structures
type IgnitionConfig struct {
	Ignition IgnitionSection `json:"ignition"`
	Passwd   PasswdSection   `json:"passwd"`
	Systemd  SystemdSection  `json:"systemd"`
}

type IgnitionSection struct {
	Version string `json:"version"`
}

type PasswdSection struct {
	Users []User `json:"users"`
}

type User struct {
	Name              string   `json:"name"`
	SSHAuthorizedKeys []string `json:"sshAuthorizedKeys"`
}

type SystemdSection struct {
	Units []SystemdUnit `json:"units"`
}

type SystemdUnit struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Contents string `json:"contents"`
}

// createIgnitionConfig creates an Ignition configuration with SSH key for core user and Docker CE installation
func createIgnitionConfig(sshPublicKey string) (string, error) {
	dockerCEServiceContents := `[Unit]
Description=Install Docker CE
Wants=network-online.target
After=network-online.target
Before=zincati.service
ConditionPathExists=!/var/lib/%N.stamp

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/curl --output-dir "/etc/yum.repos.d" --remote-name https://download.docker.com/linux/fedora/docker-ce.repo
ExecStart=/usr/bin/rpm-ostree override remove moby-engine containerd runc docker-cli --install docker-ce
ExecStart=/usr/bin/systemctl enable docker.service
ExecStart=/usr/bin/touch /var/lib/%N.stamp
ExecStart=/usr/bin/systemctl --no-block reboot

[Install]
WantedBy=multi-user.target`

	dockerTcpServiceContents := `[Unit]
Description=Forward Docker socket over TCP
After=docker.service
Requires=docker.service

[Service]
Type=simple
Restart=always
RestartSec=5
ExecStart=/usr/bin/socat TCP-LISTEN:2375,bind=0.0.0.0,fork,reuseaddr UNIX-CONNECT:/var/run/docker.sock

[Install]
WantedBy=multi-user.target`

	config := IgnitionConfig{
		Ignition: IgnitionSection{
			Version: "3.4.0",
		},
		Passwd: PasswdSection{
			Users: []User{
				{
					Name:              "core",
					SSHAuthorizedKeys: []string{strings.TrimSpace(sshPublicKey)},
				},
			},
		},
		Systemd: SystemdSection{
			Units: []SystemdUnit{
				{
					Name:     "rpm-ostree-install-docker-ce.service",
					Enabled:  true,
					Contents: dockerCEServiceContents,
				},
				{
					Name:     "docker-tcp-proxy.service",
					Enabled:  true,
					Contents: dockerTcpServiceContents,
				},
			},
		},
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ignition config: %v", err)
	}

	return string(configBytes), nil
}

// readSSHPublicKey reads the SSH public key from file
func readSSHPublicKey(publicKeyPath string) (string, error) {
	keyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH public key: %v", err)
	}
	return string(keyBytes), nil
}

// generateSSHKeyPair generates an RSA SSH key pair and saves them to the specified paths
func generateSSHKeyPair(privateKeyPath, publicKeyPath string) error {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// Validate private key
	err = privateKey.Validate()
	if err != nil {
		return fmt.Errorf("private key validation failed: %v", err)
	}

	// Create the ssh_keys directory if it doesn't exist
	keyDir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %v", err)
	}

	// Convert private key to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write private key to file
	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %v", err)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create SSH public key: %v", err)
	}

	// Format public key with comment
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	publicKeyString := fmt.Sprintf("%s coreos@container-os", strings.TrimSpace(string(publicKeyBytes)))

	// Write public key to file
	if err := ioutil.WriteFile(publicKeyPath, []byte(publicKeyString), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %v", err)
	}

	fmt.Printf("SSH key pair generated successfully:\n")
	fmt.Printf("  Private key: %s\n", privateKeyPath)
	fmt.Printf("  Public key: %s\n", publicKeyPath)

	return nil
}

func main() {

	arch := flag.String("arch", "aarch64", "Target architecture (e.g., aarch64, x86_64)")
	version := flag.String("version", "42.20250803.3.0", "Fedora CoreOS version")

	flag.Parse()

	// Construct VM image dynamically
	vmImage := fmt.Sprintf("images/coreos-%s-qemu.%s.qcow2", *version, *arch)

	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}
	fmt.Println("Checking vm assets...")
	err = ensureCoreOSImage(projectRoot, *version, *arch)
	if err != nil {
		return
	}

	// VM configuration
	memory := "2048"
	cpus := "2"
	sshPort := "2222"
	vncPort := "5900"
	dockerPort := "2375"
	sshPublicKeyPath := "ssh_keys/coreos_rsa.pub"

	// Check if VM image exists
	if _, err := os.Stat(vmImage); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: VM image '%s' not found\n", vmImage)
		os.Exit(1)
	}

	// Check if SSH public key exists, generate if needed
	sshPrivateKeyPath := "ssh_keys/coreos_rsa"
	if _, err := os.Stat(sshPublicKeyPath); os.IsNotExist(err) {
		fmt.Println("SSH keys not found. Generating new SSH key pair...")
		err := generateSSHKeyPair(sshPrivateKeyPath, sshPublicKeyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating SSH key pair: %v\n", err)
			os.Exit(1)
		}
	}

	// Read SSH public key
	sshPublicKey, err := readSSHPublicKey(sshPublicKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading SSH public key: %v\n", err)
		os.Exit(1)
	}

	// Create Ignition configuration
	ignitionConfig, err := createIgnitionConfig(sshPublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ignition config: %v\n", err)
		os.Exit(1)
	}

	// Debug: Print the generated Ignition configuration
	fmt.Println("\n=== Debug: Generated Ignition Configuration ===")
	fmt.Println(ignitionConfig)
	fmt.Println("===============================================")

	// Validate JSON by unmarshaling it back
	var testConfig IgnitionConfig
	err = json.Unmarshal([]byte(ignitionConfig), &testConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Generated JSON is invalid: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ JSON validation passed")

	// Write ignition config to temporary file for fw_cfg
	configFile := "configs/ignition-config.json"
	err = ioutil.WriteFile(configFile, []byte(ignitionConfig), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing ignition config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Ignition config written to %s (%d bytes)\n", configFile, len(ignitionConfig))
	fmt.Println("✓ Configuration file ready for fw_cfg")

	// Print connection details before starting VM
	fmt.Println("=== Fedora CoreOS VM Connection Details ===")
	fmt.Printf("VM Image: %s\n", vmImage)
	fmt.Printf("Memory: %s MB\n", memory)
	fmt.Printf("CPUs: %s\n", cpus)
	fmt.Printf("SSH Port: %s (connect with: ssh -p %s core@localhost)\n", sshPort, sshPort)
	fmt.Printf("VNC Port: %s (connect with VNC viewer to localhost:%s)\n", vncPort, vncPort)
	fmt.Printf("Docker Port: %s (Docker API accessible at localhost:%s)\n", dockerPort, dockerPort)
	fmt.Println("Docker CE: Will be installed and enabled on first boot")
	fmt.Println("Host Access: export DOCKER_HOST=tcp://localhost:2375")
	fmt.Println("==========================================")
	fmt.Println("Starting VM...")

	// Get absolute path to VM image
	absImagePath, err := filepath.Abs(vmImage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	// Build QEMU command with Ignition configuration
	qemuCmd := exec.Command("qemu-system-aarch64",
		"-M", "virt",
		"-cpu", "cortex-a72",
		"-smp", cpus,
		"-m", memory,
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", absImagePath),
		"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%s-:22,hostfwd=tcp::%s-:2375", sshPort, dockerPort),
		"-device", "virtio-net-pci,netdev=net0",
		"-vnc", fmt.Sprintf(":%s", vncPort[len(vncPort)-1:]), // Extract last digit for VNC display
		"-serial", "stdio",
		"-bios", "/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
		"-fw_cfg", fmt.Sprintf("name=opt/com.coreos/config,file=%s", configFile),
	)

	// Set up command to inherit stdout/stderr
	qemuCmd.Stdout = os.Stdout
	qemuCmd.Stderr = os.Stderr
	qemuCmd.Stdin = os.Stdin

	// Execute QEMU command
	fmt.Printf("Executing: %s\n", qemuCmd.String())
	err = qemuCmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting VM: %v\n", err)
		os.Exit(1)
	}
}
