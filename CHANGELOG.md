# Changelog

All notable Gotify MU changes are documented here.

## [0.2.1] - 2026-09-25

Managed in-app update release.

### Added

- Real administrator-managed in-app updates from **Settings → Software Update**.
- A private Gotify MU updater helper container that can build a selected official release, recreate the running Gotify MU container, verify health, and roll back automatically on failure.
- Elevated administrator API endpoints for updater status and release installation.
- Dashboard update notices now route administrators into the in-app updater instead of sending them directly to GitHub.
- Release safety checks prevent a development build from blindly downgrading itself to an older published release.
- Existing container ports, environment, restart policy, mounts, network attachments, DNS, and host mappings are preserved during replacement.
- Manual GitHub release downloads remain available as a recovery path.

### Security

- The updater helper is not published on a host port.
- Server-to-updater requests require a generated shared token.
- Starting an update requires an elevated administrator session.
- The updater helper requires Docker socket access and should only be enabled on hosts where managed self-updating is desired.

## [0.2.0] - 2026-09-25

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
- In-app release discovery for administrators with Dashboard update notices, Settings update status, and direct download links for published GitHub release assets.
- Product roadmap for MFA/2FA, LDAP/Active Directory, richer notification workflows, automation, plugin catalog, and the future Gotify MU Android client.

### Changed

- Web UI terminology now presents upstream Gotify applications as Channels.
- UI layout and styling are standardized around a Gotify MU design system.
- Light-mode surfaces use soft neutral borders rather than high-contrast outlines.
- Expired browser elevation now opens the credential/OIDC re-authentication prompt immediately instead of leaving a generic middleware/snackbar error.
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
- Published release assets and containers depend on GitHub Actions being enabled for the repository; source/Docker builds remain available as a fallback.

