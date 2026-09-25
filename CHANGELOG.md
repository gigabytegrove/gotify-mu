# Changelog

All notable Gotify MU changes are documented here.

## [0.2.0] - 2026-09-24

First formal Gotify MU pre-release.

### Added

- Shared multi-user Channels while retaining Gotify application-token compatibility.
- Global Channels with automatic assignment for existing and future users.
- Channel ownership transfer.
- Per-user realtime notification mute/unmute.
- Reversible per-user message archive and restore.
- Safe destructive-action rules for shared and Global Channels.
- Shared-channel user deletion guard until ownership is transferred.
- Experimental Chat Channels with member posting and sender identity in the Web UI.
- Redesigned Gotify MU Web UI with Dashboard, Channels, Messages, Archive, Users, Clients, Plugins, Settings, responsive navigation, and light/dark/system themes.
- Channel, message, user, client, and plugin search/filter controls.
- User display names.
- User Groups administration foundation.
- Administrative/security Audit Log foundation.
- Runtime administrator plugin installation from the Web UI.
- Persistent plugin installation under the configured plugin directory.
- Docker build identity using Gotify MU version/commit metadata.
- Product roadmap for MFA/2FA, LDAP/Active Directory, richer notification workflows, automation, plugin catalog, and the future Gotify MU Android client.

### Changed

- Web UI terminology now presents upstream Gotify applications as Channels.
- UI layout and styling are standardized around a Gotify MU design system.
- Light-mode surfaces use soft neutral borders rather than high-contrast outlines.
- Expired browser elevation is handled by returning protected UI to re-authentication instead of leaving raw middleware errors as the primary UX.
- Global Channel message deletion, history clearing, and Channel deletion are restricted to administrators.
- Shared message deletion remains per-user unless an authorized destructive action is explicitly used.

### Compatibility

- Existing Gotify application tokens remain supported.
- Existing client-token authentication remains supported.
- Existing `/application`, `/message`, and `/stream` compatibility routes remain supported.
- The official Gotify Android app remains compatible for normal receive/display workflows.
- Database migrations are additive and preserve existing users, applications, clients, messages, tokens, plugin configuration, and images.

### Known limitations

- Chat Channels are experimental and Web-only for sending. The official Gotify Android app can receive Chat Channel messages but does not provide a Gotify MU compose interface.
- MFA/2FA, passkeys, and LDAP/Active Directory are roadmap items and are not implemented in this release.
- User Groups are an administration/identity foundation in this release; group-to-Channel and group-to-policy assignment are future work.
- Audit logging is a foundation and will gain richer event categorization, detail, retention policy, and export in later releases.
- Plugin installation requires a compatible Linux Go `.so` binary built for the matching Go/Gotify MU ABI and server architecture.
- A published release container is not yet guaranteed; source/Docker builds remain the supported deployment path.

