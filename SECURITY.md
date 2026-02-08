# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.0   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please email abhishekkarki@gmail.com or open a private security advisory on GitHub.

**Please do not open public issues for security vulnerabilities.**

We will respond within 48 hours and work with you to address the issue.

## Security Best Practices

When deploying Instvisor:

1. **Run as non-root user**: The systemd service runs as the `instvisor` user
2. **Restrict filesystem access**: Use systemd `ProtectSystem=strict`
3. **Review configurations**: Ensure sensitive data is not logged
4. **Keep updated**: Regularly update to the latest version
5. **Use HTTPS**: When exposing metrics endpoints (future feature)