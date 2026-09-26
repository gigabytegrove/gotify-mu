# Changelog

All notable Gotify MU changes are documented here.

## Unreleased

- Home Assistant connections can use either a Long-Lived Access Token or the native `gotify-mu-ha` integration.
- Adds one-time native Home Assistant pairing codes, encrypted bridge credentials, inbound event routing, and outbound event delivery without a Home Assistant LLT.
- Native pairing codes expire after 15 minutes and are invalidated after successful pairing.

## [0.5.0] - 2026-09-26

Security, reliability, automation, collaboration, connector, plugin, and operations completion release.

### Security and identity

- Encrypts stored integration, connector, MFA, and other protected secrets at rest using a persistent server secret key.
- Adds login throttling and security-event auditing.
- Adds TOTP multi-factor authentication, hashed recovery codes, administrator MFA policy, and MFA-aware reauthentication.
- Adds WebAuthn/passkeys.
- Adds LDAP / Active Directory authentication alongside local authentication and OIDC.
- Adds scoped service accounts and API credentials.
- Expands session administration, lifetime policy, elevation policy, and active-session revocation.
- Redacts sensitive path/query/request values from application and audit logs.
- Hardens inbound Webhooks with HMAC signatures, replay prevention, per-route rate limits, and source CIDR restrictions.
- Stores Webhook secrets by hash for lookup while keeping the encrypted secret available only for authorized administration.
- Requires trusted/checksummed/signed native plugin installations by default; unsigned installs require an explicit administrator opt-in.

### Channels and collaboration

- Adds explicit Channel roles: Owner, Manager, Publisher, Member, and Read Only.
- Adds Group-to-Channel assignments with role and notification policy.
- Adds replies/threads, reactions, mentions, assignment, resolve/reopen state, per-user read state, attachments, message templates, and saved searches.
- Expands acknowledgement state with acknowledgement history and Channel-wide acknowledgement visibility.

### Notification automation

- Adds multi-instance automation leases to prevent duplicate scheduler execution.
- Adds schedule run history, cron schedules, excluded dates, end dates, maximum-run limits, misfire handling, and idempotent schedule delivery.
- Adds deferred Quiet Hours delivery in addition to suppression.
- Stores Digest summaries as real messages and clears queued Digest items when Digest is disabled.
- Expands Escalations with structured parent/root relationships, user/Group/Channel targets, repeat behavior, depth protection, and acknowledgement-aware cancellation.
- Adds automation-history cleanup and retention support.

### Native integrations

- Webhook Router now supports generated secret URLs, HMAC verification, replay defense, CIDR restrictions, per-source rate limits, array-aware JSON paths, conditional matching, title/message templates, explicit payload limits, and retained delivery history.
- MQTT now supports MQTT 5 and MQTT 3.1.1, QoS 0/1/2 receive flows, bounded packet sizes, custom CA certificates, mutual TLS client certificates, health/error/reconnect visibility, and connection testing.
- Home Assistant now supports connection health/error visibility, event-type filtering, entity filters, event-data filters, inbound event routing, outbound events, and connection testing.

### First-party connectors

- Adds native Email Delivery for forwarding qualifying Channel notifications through SMTP.
- Adds an SMTP Receiver with recipient routing, CIDR restrictions, optional authentication, sender/subject filters, and configurable message-size limits.
- Adds RSS / Atom monitoring with polling intervals, duplicate protection, title/category filters, and configurable priority.
- Adds a Syslog Receiver with source/facility/severity filtering and duplicate suppression.
- Adds Calendar / iCal monitoring with reminder windows, duplicate protection, title/location filters, and configurable priority.

### Plugins

- Routes plugin-created notifications through the normal Gotify MU delivery-policy engine.
- Adds plugin SHA-256 verification and Ed25519 trust/signature enforcement.
- Adds Plugin Catalog support, catalog install/update, manual verified install/update, and uninstall.
- Records plugin trust/checksum information without exposing credentials.

### Operations

- Adds administrator security-policy controls, active-session visibility/revocation, richer operations counters, diagnostics, backup creation/download, restore staging, configuration export, audit export, and retention controls.
- Expands administrative Audit Log details while recursively redacting secrets and private credentials.
- Adds cleanup for automation history, Webhook delivery history, connector seen-state, retained messages, and orphaned attachments.

### Updater and release engineering

- Managed updates verify published checksums before building a release.
- Managed update builds run the complete server test suite before replacing the current container.
- Update failures redact environment secrets from Docker error output.
- Container replacement preserves a broader set of Docker runtime settings and requires a real health check.
- Pull requests use a read-only full validation workflow with SHA-pinned Actions, UI build, lint, complete tests, production Docker build, vulnerability scanning, and SPDX SBOM generation.
- Release builds generate checksums, SBOMs, and build-provenance attestations.
- Native SMTP and Syslog receiver ports bind to loopback by default in Compose and must be deliberately exposed to a trusted network.

### Compatibility

- Existing Gotify application tokens, client tokens, REST endpoints, WebSocket delivery, normal official Gotify Android receive/display behavior, existing Channels, users, messages, archives, and compatible plugins remain supported.
- Database migration remains additive.
- Native Go plugins remain trusted in-process extensions. Signature/trust enforcement reduces untrusted-code risk but does not turn Go plugins into a sandbox.

## [0.3.0] - 2026-09-25

Native integrations and notification automation release.

### Added

- Native **Integrations** administration area.
- Inbound Webhook routes with generated URLs, JSON field mapping, default title/priority, URL regeneration, and Channel routing.
- Native MQTT broker connections with topic subscriptions, TLS support, optional credentials, automatic reconnect, and JSON message/title/priority extraction.
- Native Home Assistant event subscriptions using long-lived access tokens.
- Home Assistant event filtering and Channel routing.
- Outbound Home Assistant events, including a Web UI connection test.
- Native **Automation** administration area.
- One-time, hourly, daily, and weekly Scheduled Notifications with timezone-aware scheduling.
- Escalation rules that forward qualifying messages to another Channel when they remain unacknowledged.
- Per-user message acknowledgement and acknowledgement removal in Message History.
- Per-user Quiet Hours with timezone-aware overnight windows and priority exceptions.
- Per-user Digest delivery with configurable intervals and immediate-delivery priority.
- Additive database storage for integrations, schedules, policies, digest queues, escalation state, and acknowledgements.

### Changed

- Existing Gotify-compatible `/message` publishing now passes through the same native delivery-policy engine as Webhooks, MQTT, Home Assistant, and Scheduled Notifications.
- Quiet Hours suppress realtime delivery without removing the stored message.
- Digest mode keeps underlying messages in normal history while delaying lower-priority realtime alerts into summaries.
- Escalations stop when any Channel member acknowledges the original message.
- Integration credentials are masked after saving and are not echoed back into the Web UI.
- Native integration and automation changes are included in the administrative Audit Log.

### Compatibility

- Existing Gotify application tokens, client tokens, Android notification reception, REST routes, WebSocket delivery, Channels, users, messages, archives, and plugins remain compatible.
- Database migration is additive.

### Validation

- The v0.3.0 preview application commit `05bcd71fd8501161d6d35e41bae2dc17214f968a` passed the complete Web UI build, full `go test -v ./...` suite, complete Docker/server build, database migration startup, and green application/database health checks.
- v0.2.2 remains the published rollback baseline until v0.3.0 is accepted and formally released.
- A maintained deployment/rollback procedure is available in `docs/DEPLOYMENT.md`.

## [0.2.2] - 2026-09-25

Updater reliability and interface polish release.

### Fixed

- Fixed the Settings page repeatedly reloading after a successful update.
- Fixed update-status polling restarting immediately after every response.
- Update status now refreshes at a steady interval without hammering the server.
- Completed updates no longer trigger another page reload after the page has already reloaded.

### Added

- Overall update progress percentage.
- Clear user-facing update stages.
- Live update activity history with timestamps.
- Download progress and staged installation progress.
- Automatic recovery status shown directly in the update screen.

### Changed

- Update language now uses end-user terms instead of installation-engine terminology.
- Dashboard and Settings no longer volunteer internal build details such as commit hashes and build dates.
- Security settings show active sign-in methods only instead of unfinished roadmap features.
- Internal update diagnostics remain in server logs and are not exposed in the Web UI.

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

