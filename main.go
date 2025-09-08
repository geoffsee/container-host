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
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	VM struct {
		Architecture string `json:"architecture"`
		Version      string `json:"version"`
		Memory       string `json:"memory"`
		CPUs         string `json:"cpus"`
		Image        string `json:"image"`
		Instances    int    `json:"instances"`
	} `json:"vm"`
	Network struct {
		SSHPort        string `json:"sshPort"`
		VNCPort        string `json:"vncPort"`
		DockerPort     string `json:"dockerPort"`
		HTTPPort       string `json:"httpPort"`
		KubernetesPort string `json:"kubernetesPort"`
		K0sPort        string `json:"k0sPort"`
	} `json:"network"`
	SSH struct {
		PublicKeyPath  string `json:"publicKeyPath"`
		PrivateKeyPath string `json:"privateKeyPath"`
	} `json:"ssh"`
	QEMU struct {
		EnableAcceleration bool     `json:"enableAcceleration"`
		CustomArgs         []string `json:"customArgs"`
	} `json:"qemu"`
	Debug struct {
		PrintIgnitionConfig bool `json:"printIgnitionConfig"`
		Verbose             bool `json:"verbose"`
	} `json:"debug"`
}

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
func createIgnitionConfig(sshPublicKey string, dockerPort string) (string, error) {
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

	dockerTcpServiceContents := fmt.Sprintf(`[Unit]
Description=Forward Docker socket over TCP
After=docker.service
Requires=docker.service

[Service]
Type=simple
Restart=always
RestartSec=5
ExecStart=/usr/bin/socat TCP-LISTEN:%s,bind=0.0.0.0,fork,reuseaddr UNIX-CONNECT:/var/run/docker.sock

[Install]
WantedBy=multi-user.target`, dockerPort)
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

// calculatePort adds an offset to the base port for multiple instances
func calculatePort(basePort string, instanceOffset int) (string, error) {
	port, err := strconv.Atoi(basePort)
	if err != nil {
		return "", fmt.Errorf("invalid port number %s: %v", basePort, err)
	}
	return strconv.Itoa(port + instanceOffset), nil
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
			machine: "virt",
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

// loadConfig loads configuration from container-host.config.json with defaults
func loadConfig() (*Config, error) {
	config := &Config{}

	// Set defaults
	config.VM.Architecture = "aarch64"
	config.VM.Version = "42.20250803.3.0"
	config.VM.Memory = "4096"
	config.VM.CPUs = "4"
	config.VM.Image = ""
	config.VM.Instances = 1
	config.Network.SSHPort = "2222"
	config.Network.VNCPort = "5900"
	config.Network.DockerPort = "2377"
	config.Network.HTTPPort = "80"
	config.Network.KubernetesPort = "6443"
	config.Network.K0sPort = "9443"
	config.SSH.PublicKeyPath = "ssh_keys/coreos_rsa.pub"
	config.SSH.PrivateKeyPath = "ssh_keys/coreos_rsa"
	config.QEMU.EnableAcceleration = true
	config.QEMU.CustomArgs = []string{}
	config.Debug.PrintIgnitionConfig = true
	config.Debug.Verbose = false

	fmt.Println("=== Configuration Loading ===")

	// Check if config file exists
	configFile := "container-host.config.json"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Printf("⚠️  Config file %s not found, using default values\n", configFile)
		printConfigurationValues(config)
		return config, nil
	}

	fmt.Printf("📄 Found config file: %s\n", configFile)

	// Read and parse config file
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	fmt.Printf("✅ Successfully read config file (%d bytes)\n", len(configData))

	if err := json.Unmarshal(configData, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	fmt.Printf("✅ Successfully parsed JSON configuration\n")

	// Set VM image if not specified in config
	if config.VM.Image == "" {
		config.VM.Image = fmt.Sprintf("images/coreos-%s-qemu.%s.qcow2", config.VM.Version, config.VM.Architecture)
		fmt.Printf("💾 Generated VM image path: %s\n", config.VM.Image)
	} else {
		fmt.Printf("💾 Using configured VM image path: %s\n", config.VM.Image)
	}

	printConfigurationValues(config)
	fmt.Println("=============================")

	return config, nil
}

// printConfigurationValues displays the current configuration values
func printConfigurationValues(config *Config) {
	fmt.Println("\n📋 Current Configuration Values:")
	fmt.Printf("  VM:\n")
	fmt.Printf("    Architecture: %s\n", config.VM.Architecture)
	fmt.Printf("    Version: %s\n", config.VM.Version)
	fmt.Printf("    Memory: %s MB\n", config.VM.Memory)
	fmt.Printf("    CPUs: %s\n", config.VM.CPUs)
	fmt.Printf("    Instances: %d\n", config.VM.Instances)
	if config.VM.Image != "" {
		fmt.Printf("    Image: %s\n", config.VM.Image)
	}
	fmt.Printf("  Network:\n")
	fmt.Printf("    SSH Port: %s\n", config.Network.SSHPort)
	fmt.Printf("    VNC Port: %s\n", config.Network.VNCPort)
	fmt.Printf("    Docker Port: %s\n", config.Network.DockerPort)
	fmt.Printf("    HTTP Port: %s\n", config.Network.HTTPPort)
	fmt.Printf("    Kubernetes Port: %s\n", config.Network.KubernetesPort)
	fmt.Printf("    K0s Port: %s\n", config.Network.K0sPort)
	fmt.Printf("  SSH:\n")
	fmt.Printf("    Public Key Path: %s\n", config.SSH.PublicKeyPath)
	fmt.Printf("    Private Key Path: %s\n", config.SSH.PrivateKeyPath)
	fmt.Printf("  QEMU:\n")
	fmt.Printf("    Acceleration Enabled: %t\n", config.QEMU.EnableAcceleration)
	if len(config.QEMU.CustomArgs) > 0 {
		fmt.Printf("    Custom Args: %v\n", config.QEMU.CustomArgs)
	}
	fmt.Printf("  Debug:\n")
	fmt.Printf("    Print Ignition Config: %t\n", config.Debug.PrintIgnitionConfig)
	fmt.Printf("    Verbose: %t\n", config.Debug.Verbose)
}

func main() {
	// Load configuration first
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	arch := flag.String("arch", config.VM.Architecture, "Target architecture (e.g., aarch64, x86_64)")
	version := flag.String("version", config.VM.Version, "Fedora CoreOS version")

	flag.Parse()

	// Override config with command line arguments if provided
	if *arch != config.VM.Architecture {
		fmt.Printf("⚙️  Command-line override: Architecture changed from %s to %s\n", config.VM.Architecture, *arch)
		config.VM.Architecture = *arch
	}
	if *version != config.VM.Version {
		fmt.Printf("⚙️  Command-line override: Version changed from %s to %s\n", config.VM.Version, *version)
		config.VM.Version = *version
	}

	// Update VM image if needed
	config.VM.Image = fmt.Sprintf("images/coreos-%s-qemu.%s.qcow2", config.VM.Version, config.VM.Architecture)
	vmImage := config.VM.Image

	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}
	fmt.Println("Checking vm assets...")
	err = ensureCoreOSImage(projectRoot, *version, *arch)
	if err != nil {
		return
	}

	// Use configuration values
	memory := config.VM.Memory
	cpus := config.VM.CPUs
	sshPort := config.Network.SSHPort
	vncPort := config.Network.VNCPort
	dockerPort := config.Network.DockerPort
	httpPort := config.Network.HTTPPort
	kubernetesPort := config.Network.KubernetesPort
	k0sPort := config.Network.K0sPort
	sshPublicKeyPath := config.SSH.PublicKeyPath

	// Check if VM image exists
	if _, err := os.Stat(vmImage); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: VM image '%s' not found\n", vmImage)
		os.Exit(1)
	}

	// Check if SSH public key exists, generate if needed
	sshPrivateKeyPath := config.SSH.PrivateKeyPath
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
	ignitionConfig, err := createIgnitionConfig(sshPublicKey, dockerPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ignition config: %v\n", err)
		os.Exit(1)
	}

	// Debug: Print the generated Ignition configuration if enabled
	if config.Debug.PrintIgnitionConfig {
		fmt.Println("\n=== Debug: Generated Ignition Configuration ===")
		fmt.Println(ignitionConfig)
		fmt.Println("===============================================")
	}

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

	// Print connection details before starting VMs
	fmt.Println("=== Fedora CoreOS VM Connection Details ===")
	fmt.Printf("VM Image: %s\n", vmImage)
	fmt.Printf("Memory: %s MB per instance\n", memory)
	fmt.Printf("CPUs: %s per instance\n", cpus)
	fmt.Printf("Number of instances: %d\n", config.VM.Instances)

	for i := 0; i < config.VM.Instances; i++ {
		instanceSSHPort, _ := calculatePort(sshPort, i)
		instanceVNCPort, _ := calculatePort(vncPort, i)
		instanceHTTPPort, _ := calculatePort(httpPort, i)
		instanceDockerPort, _ := calculatePort(dockerPort, i)
		instanceKubernetesPort, _ := calculatePort(kubernetesPort, i)
		instanceK0sPort, _ := calculatePort(k0sPort, i)

		fmt.Printf("Instance %d:\n", i+1)
		fmt.Printf("  SSH Port: %s (connect with: ssh -p %s core@localhost)\n", instanceSSHPort, instanceSSHPort)
		fmt.Printf("  VNC Port: %s (connect with VNC viewer to localhost:%s)\n", instanceVNCPort, instanceVNCPort)
		fmt.Printf("  HTTP Port: %s (web services accessible at localhost:%s)\n", instanceHTTPPort, instanceHTTPPort)
		fmt.Printf("  Docker Port: %s (Docker API accessible at localhost:%s)\n", instanceDockerPort, instanceDockerPort)
		fmt.Printf("  Kubernetes API Port: %s (kubectl API at localhost:%s)\n", instanceKubernetesPort, instanceKubernetesPort)
		fmt.Printf("  K0s API Port: %s (K0s API at localhost:%s)\n", instanceK0sPort, instanceK0sPort)
		fmt.Printf("  Host Access: export DOCKER_HOST=tcp://localhost:%s\n", instanceDockerPort)
	}

	fmt.Println("Docker Engine: Will be enabled and started on first boot")
	fmt.Println("==========================================")
	fmt.Printf("Starting %d VM instance(s)...\n", config.VM.Instances)

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

	// Start multiple VM instances
	for i := 0; i < config.VM.Instances; i++ {
		// Calculate ports for this instance
		instanceSSHPort, err := calculatePort(sshPort, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating SSH port for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}
		instanceVNCPort, err := calculatePort(vncPort, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating VNC port for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}
		instanceHTTPPort, err := calculatePort(httpPort, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating HTTP port for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}
		instanceDockerPort, err := calculatePort(dockerPort, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating Docker port for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}
		instanceKubernetesPort, err := calculatePort(kubernetesPort, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating Kubernetes port for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}
		instanceK0sPort, err := calculatePort(k0sPort, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating K0s port for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}

		// Create ignition config for this instance with the correct Docker port
		instanceIgnitionConfig, err := createIgnitionConfig(sshPublicKey, instanceDockerPort)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ignition config for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}

		// Write instance-specific ignition config
		instanceConfigFile := fmt.Sprintf("configs/ignition-instance-%d.json", i+1)
		err = ioutil.WriteFile(instanceConfigFile, []byte(instanceIgnitionConfig), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing ignition config file for instance %d: %v\n", i+1, err)
			os.Exit(1)
		}

		args := []string{
			"-M", prof.machine,
			"-smp", cpus,
			"-m", memory,
			"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", absImagePath),
			"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%s-:22,hostfwd=tcp::%s-:%s,hostfwd=tcp::%s-:80,hostfwd=tcp::%s-:6443,hostfwd=tcp::%s-:9443", instanceSSHPort, instanceDockerPort, instanceDockerPort, instanceHTTPPort, instanceKubernetesPort, instanceK0sPort),
			"-device", "virtio-net-pci,netdev=net0",
			"-device", "virtio-rng-pci",
			"-vnc", fmt.Sprintf(":%s", instanceVNCPort[len(instanceVNCPort)-1:]),
			"-fw_cfg", fmt.Sprintf("name=opt/com.coreos/config,file=%s", instanceConfigFile),
			"-global", "kvm-pit.lost_tick_policy=discard",
			"-rtc", "base=utc,driftfix=slew",
		}

		// For multiple instances, only the first one gets stdio, others get null
		if i == 0 {
			args = append(args, "-serial", "stdio")
		} else {
			args = append(args, "-serial", "null")
			args = append(args, "-daemonize") // Run additional instances in background
		}

		// Add hardware acceleration if enabled in config
		if config.QEMU.EnableAcceleration {
			switch runtime.GOOS {
			case "darwin":
				args = append(args, "-accel", "hvf")
			case "linux":
				args = append(args, "-accel", "kvm")
			case "windows":
				args = append(args, "-accel", "whpx")
				// Enable Hyper-V enlightenments for better Windows guest performance
				args = append(args, "-cpu", "host,hv_relaxed,hv_spinlocks=0x1fff,hv_vapic,hv_time")
			}
		}

		// Only set -cpu if we have a profile-specific value and not on Windows with acceleration
		if prof.cpu != "" && !(runtime.GOOS == "windows" && config.QEMU.EnableAcceleration) {
			args = append([]string{"-cpu", prof.cpu}, args...)
		}

		// Prepend optional BIOS args at the end so it overrides defaults if present
		args = append(args, biosArgs...)

		// Add custom QEMU arguments from configuration
		if len(config.QEMU.CustomArgs) > 0 {
			args = append(args, config.QEMU.CustomArgs...)
			if config.Debug.Verbose {
				fmt.Printf("Added custom QEMU args for instance %d: %v\n", i+1, config.QEMU.CustomArgs)
			}
		}

		// Compose and run
		qemuCmd := exec.Command(qemuPath, args...)

		if i == 0 {
			// First instance gets full stdio
			qemuCmd.Stdout = os.Stdout
			qemuCmd.Stderr = os.Stderr
			qemuCmd.Stdin = os.Stdin
		} else {
			// Additional instances run in background
			qemuCmd.Stdout = nil
			qemuCmd.Stderr = nil
			qemuCmd.Stdin = nil
		}

		fmt.Printf("Starting instance %d: %s %s\n", i+1, qemuPath, strings.Join(args, " "))

		if i == 0 {
			// Start first instance in foreground (blocking)
			if err := qemuCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting VM instance %d: %v\n", i+1, err)
				os.Exit(1)
			}
		} else {
			// Start additional instances in background
			if err := qemuCmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting VM instance %d: %v\n", i+1, err)
				os.Exit(1)
			}
			fmt.Printf("Instance %d started in background (PID: %d)\n", i+1, qemuCmd.Process.Pid)
		}
	}
}
