# Security Policy

## Supported Versions

Security fixes are applied to the latest stable release only.

| Version | Status              |
|---------|---------------------|
| 4.x     | Actively maintained |
| < 4.0   | End of life         |

---

## Reporting a Vulnerability

Security reports are handled through **GitHub Private Security Advisories**, which ensures the issue remains confidential until a coordinated fix and disclosure can be prepared.

[**Submit a confidential report →**](https://github.com/divyo-argha/git-user/security/advisories/new)

To expedite triage, please include the following in your report:

- A clear description of the vulnerability and its potential impact
- Reproduction steps or a proof-of-concept demonstrating the issue
- The affected version(s)
- Any suggested remediation, if applicable

**Response commitments:**

- Initial acknowledgement within **48 hours**
- Coordinated patch and public disclosure for critical issues within **30 days**

---

## Supply Chain Integrity

`git-userhub` distributes pre-compiled Go binaries through platform-specific npm optional dependencies (e.g. `git-userhub-darwin-arm64`). Every release is compiled in a public, auditable GitHub Actions workflow and published to npm with **provenance attestation** via Sigstore, establishing a cryptographic link between the published tarball and the exact Git tag and CI run that produced it.

To verify the integrity of an installed version:

```bash
npm audit signatures
```

Provenance records are also available on the npm package page under the **Provenance** section for each published version: [npmjs.com/package/git-userhub](https://www.npmjs.com/package/git-userhub)

### GitHub release archives (curl installer and `git-user --update`)

- **`checksums.txt`** lists the SHA-256 of every archive. `install.sh` and `git-user --update` always verify the download against it and refuse to install on a mismatch or if it is missing.
- From the first release after v4.19.0, `checksums.txt` is also signed with keyless [cosign](https://docs.sigstore.dev/cosign/installation/) (`checksums.txt.sigstore.json`) and each archive gets a GitHub build attestation. Neither needs a signing key or a repository secret: the certificate is issued to this repository's `release.yml` workflow on a version tag. Builds use `-trimpath` and commit-derived timestamps so they can be reproduced.

To verify a download by hand:

```bash
# Signature over the checksums
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/divyo-argha/git-user/\.github/workflows/release\.yml@refs/tags/v.+$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

# Build provenance of an archive
gh attestation verify git-user_linux_x86_64.tar.gz --repo divyo-argha/git-user
```

### Dependency scanning

CI runs `govulncheck` on every push and pull request (known vulnerabilities reachable from git-user's code), Dependabot keeps Go modules and pinned GitHub Actions current, and every workflow action is pinned to a full commit SHA. A release tag must match the version in the code, or the release workflows stop.

---

## Scope

The following are considered in scope for security reports:

- Arbitrary code execution via the npm launcher script (`npm/bin/git-user.js`)
- Privilege escalation within the compiled binary
- Leakage of SSH keys, credentials, or sensitive environment data
- Supply chain attacks, including dependency confusion
- Insecure file permissions during export or import operations

The following are considered out of scope:

- Theoretical vulnerabilities without a demonstrable exploit path
- Security weaknesses in the user's local system configuration or environment
