# Security Policy

`bp` is a fintech CLI that handles API keys and can place real trades, so we take
security seriously. Thanks for helping keep the tool and its users safe.

## Supported Versions

`bitpanda-cli` is pre-1.0 and moves quickly. Only the **latest release** receives
security fixes. Please upgrade to the most recent version before reporting an issue.

| Version | Supported |
|---------|-----------|
| Latest release | ✅ |
| Older releases | ❌ |

## Reporting a Vulnerability

**Please do not open a public issue for security vulnerabilities.**

Report privately through one of these channels:

1. **GitHub Security Advisories** (preferred) — go to the
   [Security tab](https://github.com/bitpanda-labs/bitpanda-cli/security) of this
   repository and click **"Report a vulnerability"**.
2. **Email** — `security@bitpanda.com`.

Please include:

- The affected version (`bp --version`)
- Steps to reproduce
- The impact you believe the issue has

We will acknowledge your report within a few business days and keep you updated as
we investigate. We appreciate responsible disclosure and will credit reporters who
wish to be named once a fix is released.

## Scope

`bp` is a **client-side CLI**. This policy covers vulnerabilities in *this
repository's* code, for example:

- API key handling and storage (config file, environment, flags)
- TLS / transport security
- Command, argument, or output injection

Vulnerabilities in the **Bitpanda platform or Public API** itself are out of scope
here — please report those through Bitpanda's official responsible-disclosure
program.
