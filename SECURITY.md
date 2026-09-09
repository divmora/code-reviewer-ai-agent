# Security Policy & Responsible Disclosure

The **Code Reviewer AI Agent** project team takes the security of our software, dependencies, and users seriously.

---

## Supported Versions

We support the current release version with security patches:

| Version | Supported          |
| ------- | ------------------ |
| `0.1.x` | :white_check_mark: |
| `< 0.1.0`| :x:                |

---

## Reporting a Vulnerability

If you discover a security vulnerability in Code Reviewer AI Agent, please **do not open a public issue**. Instead, report it privately:

1. **Email**: Send detailed vulnerability information to `security@divmora.com`.
2. **GitHub Security Advisory**: Open a private draft security advisory at [github.com/divmora/code-reviewer-ai-agent/security/advisories/new](https://github.com/divmora/code-reviewer-ai-agent/security/advisories/new).

Please include:
- A description of the vulnerability and its potential impact.
- Steps to reproduce the issue (proof-of-concept configuration or code snippet).
- Any proposed remediation or patch.

---

## Response Timeline

- **Initial Acknowledgment**: Within 48 hours.
- **Vulnerability Assessment & Triage**: Within 5 business days.
- **Remediation & Advisory Release**: Coordinated with the reporter before public disclosure.

---

## Security Best Practices for Users

1. **Token Principle of Least Privilege**:
   - Use dedicated service accounts or Project Access Tokens.
   - For GitLab: require only `api` or `read_repository` + `write_merge_requests` scope.
   - For GitHub: require only `pull_requests: write`, `contents: read`, `statuses: write`.
   - For Bitbucket: require only `pullrequests:write`.

2. **Self-Hosted TLS / SSL Verification**:
   - Only use `--skip-tls-verify` on trusted internal private corporate networks with self-signed CAs.
   - For public or SaaS environments, always ensure valid SSL/TLS certificate verification.

3. **Read-Only LocalHarness Sandbox Policy**:
   - The agent runs with `Capabilities.RunCommand = false` and read-only static analysis safety by default, preventing arbitrary command execution during code analysis.

4. **Secret & Key Masking**:
   - Always store `CODE_REVIEWER_LITELLM_API_KEY`, `LITELLM_API_KEY`, and VCS tokens as masked/secret environment variables in CI/CD pipelines (e.g. GitLab CI Masked Variables, GitHub Secrets).
