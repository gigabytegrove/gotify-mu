<p align="center">
  <img src="assets/monita-banner.svg" alt="Monita" width="720">
</p>

# Changelog

## 1.1.8 — 2026-09-28

### Less intrusive administrator elevation

- Normal administrator pages now use the signed-in admin session instead of requiring repeated step-up elevation just to view routine administrative state.
- Viewing users, audit history, security policy, operations, sessions, service accounts, Groups, integrations, schedules, escalations, connectors, update status, and the Plugin Catalog no longer requires elevation.
- Routine non-destructive administration such as creating or editing Groups, integrations, schedules, escalations, connectors, and Channel membership/assignment settings no longer requires repeated password confirmation.
- Step-up elevation remains required for destructive or high-impact actions including user changes/deletion, Channel ownership/security changes, member/group removal, password/MFA/passkey changes, security-policy changes, backup/restore/diagnostics, session revocation, service-account credential changes, secret regeneration, software installation, and destructive deletions.
- The default administrator elevation window is now four hours instead of one hour when no explicit policy has been saved.
- The Web UI now uses the configured administration elevation duration instead of always requesting a hard-coded one-hour window.
- Local, LDAP, passkey, and OIDC elevation paths now honor the configured policy consistently, and password/OIDC requests are capped by that policy.

## 1.1.7 — 2026-09-27

### Chat image messages

- Chat Channels now support sending image attachments directly from the composer on desktop and mobile web.
- The Chat composer accepts JPEG, PNG, GIF, and WebP images, supports up to eight images per message, shows previews before sending, supports drag/drop and pasted images, and allows image-only messages.
- Image attachments are persisted before realtime delivery so recipients receive complete attachment metadata with the original message event.
- Images render inline in message history and open at full size when selected.
- Authenticated image attachment downloads are served inline while non-image attachments retain download behavior.
- Realtime message conversion now preserves extras and collaboration metadata, including attachment information.
- The MU capability document now advertises `chatImages: true` for companion clients.

## 1.1.6 — 2026-09-27

### Channel images on new and existing Channels

- Added Channel image selection directly to **Create Channel**, including local preview, change, and remove-before-create controls.
- New Channels can now be created with their image in the same workflow instead of requiring a second edit afterward.
- Existing Channels continue to expose **Upload image / Change image / Remove image** in **Edit Channel** and the Channel action menu.
- If an image upload fails after Channel creation, Monita preserves the new Channel and token and clearly directs the user to add the image from **Edit Channel**.

## 1.1.5 — 2026-09-27

### Fresh Web UI after updates

- Monita now serves the Web UI entry document with explicit no-cache headers so a completed in-app update cannot leave the browser on an older frontend bundle.
- This ensures UI changes such as the restored Channel image controls appear immediately after updating instead of requiring a hard refresh.
- Existing hashed static assets remain compatible with normal browser caching.

## 1.1.4 — 2026-09-27

### Software update status cleanup

- Completed update progress and activity are now transient instead of being shown again every time the Software Update page is reopened.
- The completion state remains visible while an update is actively being watched, then clears after the page reloads or the user navigates away and returns.
- Failed update states remain visible so errors are not silently hidden.

This changelog highlights user-visible changes in Monita. Older releases may use the previous **Gotify MU** name.

## 1.1.3 — 2026-09-27

### Channel image management

- Restored Channel image controls directly in **Edit Channel**.
- Added the current Channel image preview with **Upload image**, **Change image**, and **Remove image** actions.
- Kept Channel image shortcuts in the Channel action menu.
- Allowed Channel managers to use the same edit and image-management controls already permitted by the server.
- Kept token regeneration and destructive Channel actions restricted to owners and administrators.

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
