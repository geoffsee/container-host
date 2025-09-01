# Container OS

A Go-based tool for managing and running Fedora CoreOS virtual machines with Docker support and automated provisioning.

## Features

- 🚀 **Automated VM Provisioning**: Downloads, configures, and launches Fedora CoreOS VMs
- 🔐 **SSH Key Management**: Automatic SSH key generation and configuration
- 🐳 **Docker Integration**: Pre-configured Docker CE installation and TCP proxy
- 🔧 **Ignition Configuration**: Uses Ignition for automated system configuration
- 🖥️ **Multiple Architectures**: Supports both x86_64 and aarch64 architectures
- 📡 **Network Access**: SSH, VNC, and Docker API port forwarding

## Quick Start

### Prerequisites

- Go 1.24+
- QEMU with appropriate system emulation support
- For aarch64: EDK2 UEFI firmware (`/opt/homebrew/share/qemu/edk2-aarch64-code.fd`)

### Installation

```bash
git clone <repository-url>
cd container-os
go mod tidy
./scripts/build.sh
```

### Usage

Run with default settings (aarch64, latest stable version):

```bash
./container-os
```

Specify architecture and version:

```bash
./container-os -arch=x86_64 -version=42.20250803.3.0
```

### Connection Details

Once the VM is running, you can connect via:

- **SSH**: `ssh -p 2222 core@localhost`
- **VNC**: Connect to `localhost:5900`
- **Docker API**: `export DOCKER_HOST=tcp://localhost:2375`

## Architecture

The tool consists of several key components:

### Core Components

- **main.go**: Main application logic, VM orchestration, and QEMU execution
- **coreos_download.go**: Fedora CoreOS image management and extraction
- **configs/**: Ignition configuration files
- **ssh_keys/**: Generated SSH key pairs

### Workflow

1. **Image Management**: Downloads and caches Fedora CoreOS images
2. **Key Generation**: Creates SSH key pairs if not present
3. **Ignition Config**: Generates cloud-init style configuration
4. **VM Launch**: Starts QEMU with proper networking and storage

## Configuration

### Default Settings

| Setting | Value | Description |
|---------|-------|-------------|
| Memory | 2048 MB | VM RAM allocation |
| CPUs | 2 | Virtual CPU cores |
| SSH Port | 2222 | Host port for SSH access |
| VNC Port | 5900 | Host port for VNC access |
| Docker Port | 2375 | Host port for Docker API |

### Customization

Modify the configuration variables in `main.go` to adjust:

- Resource allocation (memory, CPUs)
- Port mappings
- SSH key paths
- VM image versions

## Docker Integration

The VM comes pre-configured with:

- **Docker CE**: Automatically installed on first boot
- **Docker TCP Proxy**: Exposes Docker socket over TCP (port 2375)
- **Host Access**: Use `export DOCKER_HOST=tcp://localhost:2375` to manage containers from the host

## File Structure

```
container-os/
├── main.go                    # Main application
├── coreos_download.go         # Image management
├── go.mod                     # Go modules
├── scripts/
│   └── build.sh              # Build script
├── configs/
│   └── ignition-config.json  # Generated Ignition config
├── ssh_keys/                  # SSH key pairs (auto-generated)
├── vms/                       # Downloaded VM images (cached)
└── images/                    # Extracted VM images
```

## Command Line Options

```bash
./container-os [OPTIONS]

Options:
  -arch string
        Target architecture (aarch64, x86_64) (default "aarch64")
  -version string
        Fedora CoreOS version (default "42.20250803.3.0")
```

## Troubleshooting

### Common Issues

**QEMU not found**: Ensure QEMU is installed and in your PATH
```bash
# macOS
brew install qemu

# Ubuntu/Debian
sudo apt install qemu-system
```

**UEFI firmware missing**: Install EDK2 firmware for aarch64
```bash
# macOS
brew install qemu
# Firmware will be at /opt/homebrew/share/qemu/edk2-aarch64-code.fd
```

**SSH connection refused**: Wait for VM to fully boot (30-60 seconds)

**Docker not accessible**: Ensure the VM has completed first-boot setup

### Debug Mode

The application outputs detailed information including:
- Download progress
- Ignition configuration validation
- VM connection details
- QEMU command execution

## Security Considerations

- SSH keys are generated with 2048-bit RSA encryption
- Docker API is exposed without TLS (development use only)
- VM network is isolated to user-mode networking

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and build
5. Submit a pull request

## License

[Add your license information here]

## Support

For issues and questions:
- Check the troubleshooting section
- Review QEMU and Fedora CoreOS documentation
- Open an issue in the project repository