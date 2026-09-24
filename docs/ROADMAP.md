# Gotify MU Roadmap

Gotify MU keeps the Gotify-compatible API and official client compatibility while building a
multi-user administration and security layer around it.

## Web UI v1

The Web UI is being rewritten around a shared Gotify MU design system rather than continuing to
extend the inherited upstream Gotify screens independently.

Planned UI areas:

- Dashboard
- Messages and Archive
- Channels and Channel details
- Users
- Clients
- Plugins
- Account and server settings
- Consistent light/dark/system themes
- Responsive desktop and mobile-web navigation
- Shared cards, tables, status chips, dialogs, empty states, and error handling

The official Gotify Android app is intentionally outside this UI rewrite and remains compatible
through the existing Gotify API.

## Authentication and account security

### MFA / 2FA

Planned local-account MFA support should include:

- TOTP authenticator applications using RFC 6238
- Recovery codes generated once and stored hashed
- Optional WebAuthn / passkeys after the TOTP foundation
- Per-user enrollment and revocation
- Administrative visibility of MFA enrollment state without exposing secrets
- Optional policy requiring MFA for administrators
- Re-authentication/elevation flows that honor MFA
- Session invalidation after password or MFA reset
- Audit events for enrollment, removal, recovery-code use, and administrative reset

MFA secrets, recovery codes, passwords, LDAP bind credentials, and other authentication secrets
must never be committed to the repository.

### LDAP / Active Directory

Planned directory authentication should support LDAP and Active Directory without replacing local
accounts as a mandatory fallback.

Initial requirements:

- LDAP and LDAPS
- StartTLS where supported
- Configurable base DN and user search/filter
- Optional service/bind account
- Direct user bind where appropriate
- Configurable username and display-name attributes
- Group membership lookup
- Mapping one or more directory groups to Gotify MU administrator access
- Local-account fallback for emergency administration
- Connection-test and configuration validation in the Web UI
- Clear authentication-source indication on each user
- No automatic deletion of local data when a directory account disappears
- Timeouts and bounded directory queries

A later phase can add automatic group-to-Channel membership mapping if there is a real need for it.

### Existing OIDC

Gotify's existing OIDC support remains supported. The longer-term authentication UI should present
Local, OIDC, and LDAP/AD as explicit authentication providers instead of scattering those settings
through unrelated configuration pages.

## Security administration

The redesigned Settings area should eventually expose:

- Authentication providers
- MFA policy
- Session policy
- Client-token policy
- Registration policy
- Security/audit events
- Server build and runtime identity

Security-sensitive actions should continue to use elevated authentication.
