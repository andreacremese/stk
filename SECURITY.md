# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in `stk`, please report it privately to help protect users.

**DO NOT** open a public GitHub issue for security vulnerabilities.

### How to Report

1. **Email**: Send details to [stk.unwind292@aleeas.com](mailto:stk.unwind292@aleeas.com)
2. **Subject**: Use "SECURITY: stk - [Brief Description]"
3. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 7 days
- **Updates**: Regular communication on progress
- **Resolution**: Aim for fix within 30 days for critical issues

### After a Fix

- Security fixes will be released as soon as possible
- A security advisory will be published after users have had time to update
- Credit will be given to the reporter (unless anonymity is requested)

## Security Best Practices

When using `stk`:

1. **Data Location**: Stack data is stored locally in SQLite
   - macOS: `~/Library/Application Support/agent-stack/stacks.db`
   - Linux: `~/.config/agent-stack/stacks.db`

2. **Sensitive Information**: Avoid storing sensitive information (passwords, API keys, PII) in stack notes

3. **File Permissions**: The database file inherits your system's default permissions. Review them if sharing systems.

4. **Updates**: Keep `stk` updated to the latest version for security fixes

## Scope

This security policy covers:
- The `stk` CLI application
- SQLite database handling
- Git integration features

Out of scope:
- Third-party dependencies (report to their respective maintainers)
- User environment configuration issues

Thank you for helping keep `stk` secure!
