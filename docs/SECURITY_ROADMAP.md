<p align="center">
  <img src="../assets/monita-banner.svg" alt="Monita" width="720">
</p>

# Monita Security Overview

Monita includes security features intended for self-hosted environments, teams, and home automation installations.

This page gives a public overview of those protections without documenting sensitive implementation details.

## Authentication

Monita supports:

- local accounts
- OIDC
- LDAP / Active Directory
- TOTP multi-factor authentication
- passkeys
- recovery codes
- service accounts for automation

Administrators can choose the authentication methods that fit their environment.

## Access control

Monita supports role-based access to Channels and administrative features.

Permissions can be assigned to users and Groups so administrators can separate viewing, publishing, management, and ownership responsibilities.

## Session security

Monita includes session controls, re-authentication for sensitive actions, active-session management, and configurable security policies.

## Secret protection

Integration credentials and other protected values are stored securely and are not shown back in plain text after they are saved.

Administrators should still protect backups, configuration files, and host access as they would for any self-hosted service.

## Integrations

Monita includes security controls for supported integrations such as Webhooks, MQTT, Home Assistant, email, and directory services.

Use encrypted transport where available and limit network access to trusted systems.

## Plugins

Plugins run with significant access to the Monita application.

Only install plugins from sources you trust.

## Updates

Monita supports managed updates and manual updates.

Keep Monita current and review release notes before major upgrades.

## Deployment recommendations

For internet-facing installations:

- use HTTPS
- use strong administrator credentials
- enable MFA
- keep Docker and the host operating system updated
- expose only the ports you actually use
- restrict administrative access where practical
- back up persistent data regularly

## Vulnerability reporting

See [SECURITY.md](../SECURITY.md) for private vulnerability reporting instructions.
