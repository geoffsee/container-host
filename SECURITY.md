# Security Policy

## Overview

container-host-cli manages system-level resources including virtual machines, network interfaces, and file systems. We take security seriously and have established this policy to ensure responsible disclosure and handling of security vulnerabilities.

## Supported Versions

Security updates are provided for the following versions:

| Version | Supported          | End of Support |
| ------- | ------------------ | -------------- |
| 0.1.x   | :white_check_mark: | TBD            |

## Security Considerations

### System-Level Access

container-host-cli operates with elevated privileges and system access:

- **QEMU Process Management**: Creates and manages QEMU virtual machine processes
- **Network Configuration**: Modifies host network settings and port forwarding
- **File System Access**: Creates and manages VM images, SSH keys, and configuration files
- **Resource Allocation**: Allocates system memory, CPU, and storage resources

### Potential Risk Areas

Users and contributors should be aware of these security-sensitive areas:

#### VM Image Handling
- Downloads and validates CoreOS images from external sources
- Manages VM disk images with potential for data exposure
- Handles Ignition configurations that may contain sensitive data

#### Network Security
- Exposes VM services on host network interfaces
- Manages SSH key pairs for VM access
- Configures Docker and Kubernetes API endpoints

#### Process Security
- Executes QEMU processes with system privileges
- Handles command-line arguments that could be exploited
- Manages temporary files and directories

#### Configuration Security
- Processes JSON configuration files that may contain sensitive settings
- Handles SSH private keys and certificates
- Manages service endpoints and authentication credentials

## Reporting Security Vulnerabilities

### DO NOT Report Security Issues Publicly

Please **DO NOT** report security vulnerabilities through public GitHub issues, discussions, or any other public forum. Public disclosure of security issues puts all users at risk.

### Responsible Disclosure Process

To report a security vulnerability:

1. **Email**: Send details to the project maintainer at the repository contact
2. **Subject Line**: Include "SECURITY: container-host-cli" in the subject
3. **Encryption**: PGP encryption encouraged (key available upon request)

### Required Information

Please include the following information in your report:

#### Basic Information
- **Vulnerability Type**: Code execution, privilege escalation, information disclosure, etc.
- **Affected Versions**: Specific versions or commit ranges affected
- **Platform Impact**: Which operating systems/architectures are affected
- **Severity Assessment**: Your assessment of the vulnerability's severity and impact

#### Technical Details
- **Description**: Clear description of the vulnerability
- **Attack Vector**: How the vulnerability can be exploited
- **Proof of Concept**: Step-by-step reproduction instructions
- **Impact Assessment**: Potential damage or data exposure
- **Suggested Fix**: Any recommendations for remediation

#### Example Report Template
```
Subject: SECURITY: container-host-cli - [Brief Description]

Vulnerability Type: [e.g., Command Injection]
Affected Versions: [e.g., v0.1.0 - current]
Platform Impact: [e.g., Linux, macOS, Windows]
Severity: [e.g., High - Remote Code Execution]

Description:
[Detailed description of the vulnerability]

Reproduction Steps:
1. [Step 1]
2. [Step 2]
3. [Observe vulnerability]

Impact:
[Description of potential damage]

Suggested Fix:
[Any recommendations for remediation]

Contact Information:
[Your preferred contact method for follow-up]
```

## Response Timeline

We are committed to addressing security vulnerabilities promptly:

- **Acknowledgment**: Within 24 hours of receipt
- **Initial Assessment**: Within 72 hours
- **Status Updates**: Weekly updates during investigation
- **Resolution Target**: 30 days for high-severity issues, 60 days for others
- **Public Disclosure**: Coordinated with reporter after fix is available

## Security Response Process

### Internal Process

1. **Triage**: Assess severity and impact
2. **Investigation**: Reproduce and analyze the vulnerability
3. **Development**: Create and test fix
4. **Testing**: Comprehensive security testing of the fix
5. **Release**: Deploy fix through normal release channels
6. **Notification**: Notify users of security update
7. **Public Disclosure**: Coordinate public disclosure with reporter

### Communication

- **Status Updates**: Regular updates to reporter on progress
- **Timeline Coordination**: Work with reporter on disclosure timeline
- **Credit**: Reporter will be credited in security advisory (unless anonymity requested)
- **CVE Assignment**: Request CVE assignment for significant vulnerabilities

## Security Best Practices for Users

### Secure Configuration

- **Network Isolation**: Run VMs in isolated network environments when possible
- **Access Control**: Limit SSH key access and rotate keys regularly
- **Resource Limits**: Set appropriate memory and CPU limits for VMs
- **Monitoring**: Monitor VM resource usage and network connections

### Operational Security

- **Regular Updates**: Keep container-host-cli updated to latest version
- **Configuration Review**: Regularly audit configuration files for sensitive data
- **Log Monitoring**: Monitor system logs for unusual QEMU or networking activity
- **Backup Security**: Secure backups of VM images and configuration data

### Development Security

- **Code Review**: Review configuration files and scripts before deployment
- **Dependency Management**: Keep dependencies updated and audit for vulnerabilities
- **Testing**: Test configurations in isolated environments before production use
- **Documentation**: Document security-relevant configuration choices

## Security Hardening Recommendations

### System-Level Hardening

```bash
# Restrict file permissions for SSH keys
chmod 600 ssh_keys/container-host-*

# Limit QEMU process capabilities
# Configure system-level resource limits

# Monitor network connections
netstat -tulpn | grep qemu
```

### Configuration Hardening

```json
{
  "security": {
    "restrictNetworkAccess": true,
    "enableFirewall": true,
    "limitResourceUsage": true
  },
  "qemu": {
    "enableSeccomp": true,
    "restrictDeviceAccess": true
  }
}
```

## Known Security Limitations

### Current Limitations

- **Privilege Requirements**: Requires elevated privileges for VM management
- **Network Exposure**: VM services are exposed on host network by default
- **Image Integrity**: Limited cryptographic verification of downloaded images
- **Process Isolation**: QEMU processes run with host-level access

### Mitigation Strategies

- Use containerization or sandboxing to limit host access
- Implement network-level access controls and firewalls
- Verify image checksums and signatures when available
- Run in dedicated, isolated environments for production use

## Compliance and Legal Considerations

### Regulatory Compliance

This tool may be subject to various compliance requirements depending on usage:

- **GDPR**: If processing EU personal data in VMs
- **HIPAA**: If handling healthcare data
- **SOX**: If used in financial reporting systems
- **Export Controls**: Software may be subject to export regulations

### Legal Disclaimers

- **No Warranty**: Security features provided "as is" without warranty
- **User Responsibility**: Users are responsible for secure configuration and operation
- **Compliance**: Users must ensure compliance with applicable regulations
- **Liability**: See [LICENSE](LICENSE) for liability limitations

## Security Contact Information

For security-related inquiries:
- **GitHub Security**: Use GitHub's private vulnerability reporting feature
- **Repository Issues**: For general security questions (non-sensitive)
- **Response Hours**: Best effort response within 24-72 hours

## Security Advisory History

Security advisories will be published at:
- GitHub Security Advisories: https://github.com/geoffsee/container-host/security/advisories
- Project Security Page: https://github.com/geoffsee/container-host/blob/master/SECURITY.md

---

**Important Legal Notice**: This security policy provides general guidance but does not constitute legal advice or warranty. Users should consult qualified security professionals and legal counsel for specific security and compliance requirements.

**Last Updated**: September 2025