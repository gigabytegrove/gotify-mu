# Gotify MU Security Status

This document tracks the security controls implemented in the Gotify MU v0.5 preview and the remaining architectural boundaries administrators should understand.

## Authentication providers

Gotify MU supports:

- local username/password authentication
- OIDC
- LDAP / Active Directory
- Gotify client tokens
- Gotify application tokens
- scoped Gotify MU service-account credentials

Authentication providers can coexist during migration.

## Multi-factor authentication

Implemented:

- TOTP authenticator applications
- hashed one-time recovery codes
- per-user enrollment/removal
- administrator enrollment visibility
- administrator reset workflows
- optional MFA requirement for administrators
- optional MFA requirement for local users
- MFA-aware elevated/re-authentication flows
- WebAuthn/passkeys

MFA secrets and recovery material are never returned after their intended enrollment/recovery step and are excluded from ordinary logs/audit details.

## LDAP / Active Directory

Implemented:

- LDAP and LDAPS
- service/bind credentials
- configurable user search base/filter
- configurable display-name and group attributes
- directory group-to-admin policy
- optional allowed-user group
- auto-registration
- optional username linking
- custom CA support
- bounded operations/timeouts
- connection/authentication diagnostics
- local/OIDC coexistence

TLS certificate verification is enabled by default. The insecure-skip-verify option exists only as an explicit diagnostic escape hatch and should not be used in normal deployments.

## Password and session policy

Administrator policy includes:

- configurable minimum password length
- bcrypt password hashing
- configurable session inactivity
- configurable elevated-session lifetime
- active-session visibility
- administrator session revocation
- login throttling
- MFA policy

Session cookies are HttpOnly and SameSite=Strict. Deployments served over HTTPS should enable Secure cookies.

## Stored secrets

Protected Gotify MU secrets are encrypted at rest using AES-GCM with a persistent 32-byte server key.

The key can be supplied by:

- `GOTIFY_MU_SECRET_KEY`
- `GOTIFY_MU_SECRET_KEY_FILE`

If neither is supplied, Gotify MU creates a private key file in persistent data with mode 0600.

The encryption key must be included in disaster-recovery planning. A database backup containing encrypted values is not sufficient by itself if the corresponding encryption key is lost.

## Logging and audit safety

Implemented:

- sensitive query/path value redaction
- Webhook secret path masking
- updater Docker argument/environment redaction
- recursive redaction of password/token/secret/private-key/API-key fields from administrative request summaries
- successful administrative mutation audit events
- login/security-event auditing
- audit export and configurable retention

Audit details intentionally avoid full Webhook payload storage.

## Webhook security

Inbound Webhooks can use:

- high-entropy generated secret URLs
- encrypted secret storage
- hashed secret lookup
- HMAC-SHA256 request signatures
- timestamp validation
- replay protection
- source CIDR restrictions
- rate limiting
- explicit payload-size limits
- retained outcome history without payload/credential retention

## Plugin trust boundary

Native Gotify-compatible Go plugins execute inside the Gotify MU process. They must therefore be treated as trusted server code.

v0.5 reduces plugin supply-chain risk through:

- SHA-256 verification
- Ed25519 signatures
- administrator-configured trusted public keys
- unsigned installation disabled by default
- HTTPS-only catalog/download requirements
- verified update staging

These controls establish authenticity/integrity. They do **not** sandbox native Go code.

An isolated out-of-process extension protocol may be considered in a later release for integrations that should not be trusted with in-process execution.

## Updater trust boundary

The managed updater requires Docker socket access and therefore has host-level container-management authority.

Mitigations include:

- no published updater host port
- private shared server/updater token
- release checksum verification
- full server test suite during release builds
- sensitive Docker argument redaction
- runtime configuration preservation
- required application health verification
- automatic container rollback on replacement failure

The updater should only be enabled where managed in-app Docker updates are desired.

## CI and release integrity

v0.5 uses:

- read-only pull-request workflow permissions
- SHA-pinned GitHub Actions
- lint and full tests
- production Docker build with tests enabled
- filesystem/dependency vulnerability scanning
- container vulnerability scanning
- SPDX SBOM generation
- release checksums
- build-provenance attestation

## Recovery

Security-policy changes must not silently remove the final usable administrator authentication path.

Backups must preserve:

- database/application data
- the Gotify MU encryption key
- relevant TLS/certificate material
- external authentication configuration

See `docs/DEPLOYMENT.md` for deployment and rollback procedures.
