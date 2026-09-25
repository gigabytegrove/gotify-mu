# Gotify MU Security Roadmap

This document tracks authentication and account-security work that is intentionally separate from the Web UI rewrite.

## Goals

Gotify MU should support stronger authentication without breaking existing Gotify client-token, application-token, or API compatibility.

The security work should be additive and migration-safe. Existing local accounts must continue to work unless an administrator explicitly changes authentication policy.

## Planned authentication capabilities

### Multi-factor authentication

Initial MFA work should focus on standards that do not require a proprietary service:

- TOTP authenticator apps
- recovery codes
- per-user MFA enrollment and removal
- administrator visibility into enrollment status
- optional administrator-enforced MFA policy
- step-up authentication for destructive administrative operations

A later phase may add WebAuthn/passkeys as a stronger phishing-resistant option.

MFA secrets and recovery codes must never be returned after initial enrollment and must never be written to logs.

### LDAP / Active Directory

Directory authentication should be implemented as a server authentication provider rather than replacing Gotify MU's authorization model.

Expected configuration includes:

- LDAP or LDAPS server URI
- bind DN/service account support
- configurable user search base and filter
- username and display-name attributes
- optional group-to-admin mapping
- TLS certificate validation controls
- connection and authentication diagnostics
- explicit timeout behavior
- optional local-account fallback

Directory passwords must never be stored by Gotify MU.

### Existing providers

Gotify MU should continue to support:

- local username/password authentication
- OIDC
- Gotify client tokens
- Gotify application tokens

Authentication providers should be selectable independently so an administrator can run local + LDAP, local + OIDC, or another supported combination during migration.

## Security policy controls

Future administrative settings should support:

- require MFA for administrators
- require MFA for all local users
- disable local password login after external authentication is validated
- session lifetime
- elevated-session lifetime
- account lockout / rate limiting
- trusted proxy awareness for authentication logs
- security-event audit log

## Recovery and migration

Authentication changes must not create an easy administrator lockout path.

Before enabling a policy that could disable the final usable administrator login, Gotify MU should verify that another usable administrative authentication method exists.

Recovery codes and emergency local-admin recovery should be designed before MFA can be globally enforced.

## UI direction

The Web UI should eventually expose authentication under **Settings → Security** with separate sections for:

- Authentication providers
- MFA policy
- User enrollment status
- Sessions
- Audit / security events

The current UI may show these features as planned, but must not imply they are active before the corresponding backend support exists.
