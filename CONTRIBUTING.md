# Contributing to container-host-cli

Thank you for your interest in contributing to container-host-cli! This document provides guidelines and information for contributors to ensure smooth collaboration while protecting the project's legal integrity.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Process](#development-process)
- [Contribution Guidelines](#contribution-guidelines)
- [Legal Requirements](#legal-requirements)
- [Security Considerations](#security-considerations)
- [Community and Support](#community-and-support)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors must follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- **Go**: Version 1.24 or later
- **Bun**: For building the Node.js wrapper
- **QEMU**: For testing VM functionality
- **Git**: For version control
- **Platform-specific acceleration**: KVM (Linux), HVF (macOS), or WHPX (Windows)

### Development Setup

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/container-host.git
   cd container-host
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/geoffsee/container-host.git
   ```
4. Install dependencies and build:
   ```bash
   bun install
   bun run build
   ```

### Testing Your Setup

Verify your development environment:

```bash
# Build the project
bun run build

# Run basic functionality tests
make run

# Test with custom configuration
./container-host -arch x86_64 -version 42.20250803.3.0
```

## Development Process

### Branch Strategy

- `master`: Stable, release-ready code
- `develop`: Integration branch for features (if exists)
- `feature/description`: New features or enhancements
- `bugfix/description`: Bug fixes
- `security/description`: Security-related fixes (private until resolved)

### Workflow

1. **Create an Issue**: For significant changes, create an issue first to discuss the approach
2. **Branch Creation**: Create a feature branch from `master`
3. **Development**: Make your changes following our coding standards
4. **Testing**: Ensure all tests pass and add new tests for new functionality
5. **Documentation**: Update documentation as needed
6. **Pull Request**: Submit a PR with clear description and testing evidence

## Contribution Guidelines

### Code Standards

#### Go Code Standards
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comprehensive comments for public functions
- Handle errors appropriately with context
- Write unit tests for new functionality

#### Node.js Wrapper Standards
- Follow TypeScript best practices
- Use consistent error handling patterns
- Maintain compatibility across supported Node.js versions
- Update type definitions as needed

### Commit Message Format

Use clear, descriptive commit messages:

```
type(scope): brief description

Longer description if needed, explaining:
- What changed
- Why it changed  
- Any breaking changes or migration notes
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### Documentation Requirements

All contributions must include appropriate documentation:

- **Code Comments**: For all public functions and complex logic
- **README Updates**: For new features or changed behavior
- **Configuration Documentation**: For new config options
- **Security Considerations**: For changes affecting system security

### Testing Requirements

#### Required Tests
- **Unit Tests**: For all new functions and methods
- **Integration Tests**: For VM management and networking features
- **Cross-Platform Tests**: Verify functionality on supported platforms
- **Security Tests**: For any security-sensitive changes

#### Test Execution
```bash
# Run all tests
make test

# Platform-specific testing
make test-linux
make test-macos
make test-windows

# Security testing
make security-test
```

## Legal Requirements

### Contributor License Agreement

By contributing to this project, you agree that:

1. **Ownership**: You have the right to contribute the code
2. **License Grant**: You grant the project maintainers a perpetual, worldwide, non-exclusive, royalty-free license to use, reproduce, modify, and distribute your contributions
3. **MIT Compatibility**: Your contributions are compatible with the project's MIT license
4. **No Warranty**: Your contributions are provided "as is" without warranty

### Intellectual Property

- **Original Work**: Only contribute original work or properly attributed open-source code
- **Dependencies**: New dependencies must be compatible with MIT licensing
- **Third-Party Code**: Clearly mark and attribute any third-party code with proper licenses
- **Patent Rights**: Ensure you have the right to contribute any patented technologies

### Sensitive Information

Never include in contributions:
- Personal credentials or API keys
- Proprietary code from other projects
- Customer data or private information
- Security vulnerabilities (report these privately via [SECURITY.md](SECURITY.md))

## Security Considerations

### VM and System Security

Given that this tool manages virtual machines and system resources:

- **Input Validation**: Rigorously validate all user inputs
- **Command Injection**: Prevent command injection in QEMU arguments
- **File System Access**: Limit file system access to necessary directories
- **Network Security**: Validate network configuration parameters
- **Resource Limits**: Implement appropriate resource constraints

### Secure Coding Practices

- **Error Handling**: Don't leak sensitive information in error messages
- **Logging**: Avoid logging sensitive configuration data
- **File Permissions**: Ensure generated files have appropriate permissions
- **Temporary Files**: Securely handle temporary files and cleanup

### Vulnerability Reporting

If you discover security vulnerabilities:
1. **DO NOT** create a public issue
2. Follow the process outlined in [SECURITY.md](SECURITY.md)
3. Allow time for assessment and patching before public disclosure

## Community and Support

### Getting Help

- **GitHub Issues**: For bug reports and feature requests
- **Discussions**: For general questions and community support
- **Documentation**: Check existing docs before asking questions

### Review Process

All contributions go through code review:

1. **Automated Checks**: CI/CD pipeline validates code quality and tests
2. **Security Review**: Security-sensitive changes receive additional scrutiny
3. **Maintainer Review**: Core maintainers review for code quality and design
4. **Community Input**: Community members may provide feedback on significant changes

### Response Times

We aim to respond to contributions within:
- **Issues**: 48 hours for initial response
- **Pull Requests**: 5 business days for initial review
- **Security Issues**: 24 hours for acknowledgment

## Legal Disclaimers

### Limitation of Liability

Contributors acknowledge that:

- This software manages system-level resources including virtual machines
- Improper use may cause system instability or data loss
- Contributors are not liable for damages resulting from use of their contributions
- The project maintainers reserve the right to reject contributions that increase legal or security risks

### Export Compliance

This project may be subject to export control regulations. Contributors must ensure their contributions comply with applicable export control laws in their jurisdiction.

### Trademark Notice

"container-host-cli" and associated trademarks are property of Geoff Seemueller. Contributors do not acquire trademark rights through their contributions.

---

## Contact

For legal questions or concerns about contributions, contact:
- Project Maintainer: Geoff Seemueller
- Security Issues: See [SECURITY.md](SECURITY.md) for procedure
- General Questions: Create a GitHub issue