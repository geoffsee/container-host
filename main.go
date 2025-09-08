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
	"runtime"
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

// createIgnitionConfig creates an Ignition configuration with SSH key for core user and Docker engine setup
func createIgnitionConfig(sshPublicKey string) (string, error) {
	setupLinger := `[Unit]
Description=Enable linger for user 'core' (start user manager at boot)
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/loginctl enable-linger core

[Install]
WantedBy=multi-user.target
`
	dockerServiceContents := `[Unit]
Description=Enable and start Docker engine
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/systemctl enable docker.service
ExecStart=/usr/bin/systemctl start docker.service

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
ExecStart=/usr/bin/socat TCP-LISTEN:2377,bind=0.0.0.0,fork,reuseaddr UNIX-CONNECT:/var/run/docker.sock

[Install]
WantedBy=multi-user.target`
	disableZincatiServiceContents := `[Unit]
Description=Disable Zincati automatic updates
DefaultDependencies=no

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/systemctl mask zincati.service

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
					Name:     "docker-setup.service",
					Enabled:  true,
					Contents: dockerServiceContents,
				},
				{
					Name:     "docker-tcp-proxy.service",
					Enabled:  true,
					Contents: dockerTcpServiceContents,
				},
				{
					Name:     "disable-zincati.service",
					Enabled:  true,
					Contents: disableZincatiServiceContents,
				},
				{Name: "setup-linger-core.service", Enabled: true, Contents: setupLinger},
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
	publicKeyString := fmt.Sprintf("%s coreos@container-host", strings.TrimSpace(string(publicKeyBytes)))

	// Write public key to file
	if err := ioutil.WriteFile(publicKeyPath, []byte(publicKeyString), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %v", err)
	}

	fmt.Printf("SSH key pair generated successfully:\n")
	fmt.Printf("  Private key: %s\n", privateKeyPath)
	fmt.Printf("  Public key: %s\n", publicKeyPath)

	return nil
}

// findFirstExisting returns the first path that exists, or empty string.
func findFirstExisting(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

type qemuProfile struct {
	binary         string
	machine        string
	cpu            string
	biosCandidates []string
}

func profileForArch(arch string) qemuProfile {
	switch arch {
	case "aarch64":
		return qemuProfile{
			binary:  "qemu-system-aarch64",
			machine: "virt",
			cpu:     "max",
			biosCandidates: []string{
				"/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
				"/usr/share/edk2/aarch64/QEMU_EFI.fd",
				"/usr/share/AAVMF/AAVMF_CODE.fd",
				"/usr/share/qemu-efi-aarch64/QEMU_EFI.fd",
			},
		}
	case "x86_64", "amd64":
		return qemuProfile{
			binary:  "qemu-system-x86_64",
			machine: "q35",
			cpu:     "max",
			biosCandidates: []string{
				"/opt/homebrew/share/qemu/edk2-x86_64-code.fd",
				"/usr/share/OVMF/OVMF_CODE.fd",
				"/usr/share/edk2/ovmf/OVMF_CODE.fd",
			},
		}
	default:
		// Fallback: best effort — QEMU naming usually matches qemu-system-<arch>.
		return qemuProfile{
			binary:         fmt.Sprintf("qemu-system-%s", arch),
			machine:        "virt",
			cpu:            "host",
			biosCandidates: []string{
				// no good generic candidates — let QEMU default if none exist
			},
		}
	}
}

// isWindowsHost checks if the current host OS is Windows
func isWindowsHost() bool {
	return runtime.GOOS == "windows"
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
	memory := "4096"
	cpus := "4"
	sshPort := "2222"
	vncPort := "5900"
	dockerPort := "2377"
	httpPort := "80"
	kubernetesPort := "6443"
	k0sPort := "9443"
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
	configFile := "configs/ignition.json"
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
	fmt.Printf("HTTP Port: %s (web services accessible at localhost:%s)\n", httpPort, httpPort)
	fmt.Printf("Docker Port: %s (Docker API accessible at localhost:%s)\n", dockerPort, dockerPort)
	fmt.Printf("Kubernetes API Port: %s (kubectl API at localhost:%s)\n", kubernetesPort, kubernetesPort)
	fmt.Println("Docker Engine: Will be enabled and started on first boot")
	fmt.Println("Host Access: export DOCKER_HOST=tcp://localhost:2377")
	fmt.Println("==========================================")
	fmt.Println("Starting VM...")

	// Get absolute path to VM image
	absImagePath, err := filepath.Abs(vmImage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	prof := profileForArch(*arch)

	// Ensure the QEMU binary exists in PATH
	qemuPath, err := exec.LookPath(prof.binary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: required QEMU binary %q not found in PATH (arch=%s)\n", prof.binary, *arch)
		os.Exit(1)
	}

	// Optional: choose a matching firmware/BIOS if present
	biosPath := findFirstExisting(prof.biosCandidates...)
	biosArgs := []string{}
	if biosPath != "" {
		biosArgs = []string{"-bios", biosPath}
	}

	args := []string{
		"-M", prof.machine,
		"-smp", cpus,
		"-m", memory,
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", absImagePath),
		"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%s-:22,hostfwd=tcp::%s-:2377,hostfwd=tcp::%s-:80,hostfwd=tcp::%s-:6443,hostfwd=tcp::%s-:9443", sshPort, dockerPort, httpPort, kubernetesPort, k0sPort),
		"-device", "virtio-net-pci,netdev=net0",
		"-device", "virtio-rng-pci",
		"-vnc", fmt.Sprintf(":%s", vncPort[len(vncPort)-1:]),
		"-serial", "stdio",
		"-fw_cfg", fmt.Sprintf("name=opt/com.coreos/config,file=%s", configFile),
	}

	switch runtime.GOOS {
	case "darwin":
		args = append(args, "-accel", "hvf")
	case "linux":
		args = append(args, "-accel", "kvm")
	}

	// Add Windows virtualization features when running on Windows host
	if isWindowsHost() {
		// Enable hardware acceleration for Windows
		args = append(args, "-accel", "whpx")
		// Enable Hyper-V enlightenments for better Windows guest performance
		args = append(args, "-cpu", "host,hv_relaxed,hv_spinlocks=0x1fff,hv_vapic,hv_time")
	}

	// Only set -cpu if we have a profile-specific value
	if prof.cpu != "" {
		args = append([]string{"-cpu", prof.cpu}, args...)
	}

	// Prepend optional BIOS args at the end so it overrides defaults if present
	args = append(args, biosArgs...)

	// Compose and run
	qemuCmd := exec.Command(qemuPath, args...)
	qemuCmd.Stdout = os.Stdout
	qemuCmd.Stderr = os.Stderr
	qemuCmd.Stdin = os.Stdin

	fmt.Printf("Executing: %s %s\n", qemuPath, strings.Join(args, " "))
	if err := qemuCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting VM: %v\n", err)
		os.Exit(1)
	}
}
