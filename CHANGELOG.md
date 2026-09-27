<p align="center">
  <img src="assets/monita-banner.svg" alt="Monita" width="720">
</p>

# Changelog

This changelog highlights user-visible changes in Monita. Older releases may use the previous **Gotify MU** name.

## 1.1.2 — 2026-09-27

### Single-container software updates

- Software updates now run entirely inside the main Monita container.
- Removed the updater worker container and Docker socket requirement.
- Monita downloads the published runtime for the server architecture, verifies its checksum and version, installs it into persistent Monita data, and restarts itself in place.
- Updated runtimes survive normal container restarts because the active runtime is stored with Monita's persistent data.
- Fresh installations and manual upgrades can continue using prebuilt container images; no local compilation is required.

## 1.1.1 — 2026-09-27

### Faster installs and updates

- Normal Docker installations now download a prebuilt Monita image instead of compiling Monita on the server.
- Prepared the prebuilt release-image distribution used by later update improvements.
- Local source builds remain available for development and troubleshooting.
- Improved container publishing for common 64-bit Intel/AMD and ARM systems.

## 1.1.0 — 2026-09-27

### Monita rebrand

- Renamed the product from Gotify MU to **Monita**
- Added the new Monita logo, icon, colors, and product identity
- Updated the Web UI, browser/PWA identity, documentation, release presentation, and Docker naming
- Kept Gotify compatibility for supported clients and integrations
- Preserved existing users, Channels, messages, tokens, and persistent data during the transition

### Updates and deployment

- Standardized the Docker deployment around the Monita name
- Simplified the in-app update experience
- Continued support for existing installations during the transition

## 1.0.2 — 2026-09-27

### Improved

- Simplified the Software Update page
- Reduced update-related clutter in the UI
- Simplified the Docker deployment model

## 1.0.1 — 2026-09-27

### Fixed

- Fixed an issue that could prevent preview or development installations from moving to a published release
- Improved update handling for locally built installations

## 1.0.0 — 2026-09-26

First stable release of the multi-user platform.

### Added

- native Home Assistant pairing
- Chat Channel improvements
- typing indicators for compatible clients
- improved compatibility discovery
- expanded testing and release validation

### Compatibility

- continued support for normal Gotify notification delivery
- continued support for existing application and client tokens

## 0.5.0 — 2026-09-26

Major platform expansion focused on security, collaboration, automation, integrations, plugins, and administration.

### Security and identity

- MFA and passkeys
- LDAP / Active Directory
- service accounts
- session administration
- audit logging
- stronger protection for saved integration credentials

### Channels and collaboration

- Channel roles
- Group assignments
- replies and threads
- reactions
- mentions
- acknowledgements
- assignments
- resolve/reopen
- attachments
- templates
- saved searches

### Automation

- expanded scheduling
- Quiet Hours
- Digests
- escalation workflows
- automation history

### Integrations

- improved Webhooks
- improved MQTT
- improved Home Assistant support
- email delivery and inbound email
- RSS / Atom
- Syslog
- Calendar / iCal

### Plugins and operations

- Plugin Catalog improvements
- plugin verification
- backup and restore tools
- diagnostics
- retention and cleanup controls

## 0.3.0 — 2026-09-25

### Added

- native Integrations administration
- Webhook routing
- MQTT
- Home Assistant
- scheduled notifications
- escalation rules
- message acknowledgements
- Quiet Hours
- Digests

## 0.2.2 — 2026-09-25

### Fixed

- improved update-status refresh behavior
- improved update progress and activity display
- clearer update language

## 0.2.1 — 2026-09-25

### Added

- administrator-managed in-app updates
- update notices in the Dashboard and Settings
- safer update handling for development builds

## 0.2.0 — 2026-09-25

First formal pre-release.

### Added

- shared multi-user Channels
- Global Channels
- Channel ownership transfer
- per-user notification preferences
- archive and restore
- Chat Channels
- redesigned Web UI
- user management
- Groups foundation
- Audit Log foundation
- plugin installation
- in-app release discovery

### Compatibility

- existing Gotify application tokens remain supported
- existing client-token authentication remains supported
- normal Gotify-compatible notification delivery remains supported
