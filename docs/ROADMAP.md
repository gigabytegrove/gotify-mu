# Gotify MU Product Roadmap

Gotify MU is the permanent product name. The project keeps Gotify protocol/API compatibility where it is useful, but new Gotify MU capabilities are allowed to move beyond upstream Gotify's original scope.

The official Gotify Android app remains supported for normal receive-only notification workflows. A Gotify MU Android fork is planned for features that require client-side support such as Chat Channels, replies, acknowledgements, attachments, and richer interaction.

## Current foundation

Implemented or in active development:

- Multi-user/shared Channels
- Global Channels
- Channel ownership transfer
- Per-user notification mute
- Per-user archive/restore
- Global Channel destructive-action protections
- Experimental server/Web Chat Channels
- Redesigned Gotify MU Web UI
- Runtime plugin upload/install from the Web UI
- User display names
- User Groups foundation
- Administrative/security audit log foundation
- Existing Gotify local auth, OIDC, client tokens, application tokens, REST and WebSocket compatibility

## Identity, authorization, and security

### User identity

Planned:

- display names
- avatars
- user profiles
- authentication-source indication
- service accounts
- scoped API credentials
- user Groups
- group-to-Channel assignment
- group-to-policy assignment
- channel roles: Owner, Manager, Publisher, Member, Read Only
- explicit permission checks instead of inferring all permissions from ownership/admin state

### MFA / 2FA

Planned local-account MFA:

- TOTP authenticator applications using RFC 6238
- one-time recovery codes stored hashed
- enrollment/revocation
- administrator enrollment visibility
- administrator reset workflow
- require MFA for administrators
- require MFA for all local users
- MFA-aware elevation/re-authentication
- session invalidation after password/MFA reset
- WebAuthn/passkeys after TOTP is stable

Secrets, passwords, recovery codes, LDAP bind credentials, and private authentication material must never be written to logs or committed to the repository.

### LDAP / Active Directory

Planned:

- LDAP
- LDAPS
- StartTLS
- service/bind account or direct-bind modes
- configurable search base/filter
- configurable username/display-name attributes
- directory group lookup
- directory-group-to-admin mapping
- optional directory-group-to-Gotify-MU-group mapping
- bounded queries/timeouts
- certificate validation
- connection diagnostics in the Web UI
- local emergency-admin fallback

OIDC remains a first-class supported provider.

### Security administration

Planned:

- authentication-provider UI
- MFA policy
- session lifetime
- elevated-session lifetime
- account lockout/rate limiting
- active-session visibility/revocation
- security event audit log
- trusted-proxy-aware source information
- server-side encryption key management for stored authentication secrets

## Channels and communication

### Channel permissions

Planned:

- Owner
- Manager
- Publisher
- Member
- Read Only
- role-based member management
- role-aware destructive actions
- group assignment
- per-role publishing permissions

### Interactive messages and Chat Channels

Server support begins with the existing experimental Chat Channel capability. Full implementation requires the Gotify MU Android client.

Planned:

- mobile compose/send
- replies
- lightweight threads
- reactions
- @mentions
- unread/read state
- message acknowledgement
- assignment/"I'm handling this"
- resolve/reopen state
- attachments
- inline images
- richer sender identity
- channel topic/description presentation

The goal is **alerts + collaboration**, not a general-purpose Slack replacement.

## Notification intelligence

These are native Gotify MU capabilities, not plugins, because they directly affect message delivery behavior.

Planned:

- user quiet hours
- per-Channel quiet hours
- priority-based quiet-hour exceptions
- hold, suppress, or defer behavior
- timezone-aware quiet-hour schedules
- per-Channel priority thresholds
- per-device notification preferences
- snooze
- one-time scheduled notifications
- recurring scheduled notifications
- per-Channel schedules
- schedule enable/disable and history
- digest mode
- hourly, daily, and custom digest schedules
- per-user and per-Channel digest rules
- immediate-delivery exceptions for higher-priority messages
- escalation rules
- multi-stage escalation paths
- acknowledgement/resolution-based escalation
- acknowledgement deadlines
- fallback/escalation targets
- user, Group, and Channel escalation targets
- repeat-until-acknowledged
- stop escalation after acknowledgement or resolution
- notification templates

## Rich messages

Planned:

- attachments
- images
- action buttons
- canonical URLs
- structured fields
- improved Markdown
- message templates
- reusable message layouts
- safe rendering/sanitization
- attachment retention controls

## Automation and integrations

### Native integration framework

The following integrations are planned as built-in Gotify MU functionality rather than plugins because they are foundational notification transports or are expected to participate deeply in routing and delivery behavior.

#### Webhook Router

- named inbound webhook endpoints
- generic outbound Webhooks
- route incoming requests to one or more Channels
- routing rules
- transformation rules
- conditional rules
- reusable endpoints
- templates
- retry policy
- delivery history and errors
- webhook signing/secrets

#### MQTT

- connect to one or more MQTT brokers
- subscribe to configured topics
- route topic events into Channels
- topic filters
- payload templates
- optional publishing from Gotify MU where appropriate
- per-connection health/status

#### Home Assistant

- direct Home Assistant integration
- receive selected Home Assistant events and notifications
- route selected events into Gotify MU Channels
- send supported Gotify MU actions/events back to Home Assistant
- configurable entity and event filters
- connection/status visibility in the Web UI

The native integration framework should share consistent configuration, health/status, secrets handling, routing, retry, and audit behavior.

## Plugin ecosystem

Plugins are intended for optional external sources and specialized integrations that do not need to control Gotify MU's core message-delivery behavior.

Current:

- plugin list/configuration
- enable/disable
- administrator upload/install from the Web UI

### Approved first-party plugins

- **Email Gateway**
  - connect to supported mailboxes
  - turn matching incoming email into Channel messages
  - rules by sender, recipient, subject, and mailbox
  - attachment handling where practical

- **SMTP Receiver**
  - receive SMTP directly from devices and services
  - intended for systems that can only send email alerts
  - recipient-to-Channel routing
  - sender and subject rules
  - configurable size and attachment limits

- **RSS / Atom Monitor**
  - monitor RSS and Atom feeds
  - post newly discovered entries to Channels
  - per-feed polling interval
  - duplicate protection
  - optional title/category filtering

- **Syslog Receiver**
  - receive syslog messages
  - source/facility/severity filtering
  - route matching events into Channels
  - duplicate/noise controls

- **Calendar / iCal**
  - subscribe to iCal-compatible calendars
  - notify before upcoming events
  - per-calendar and per-event reminder rules
  - duplicate-event protection

### Under consideration

These are not committed roadmap items yet:

- **Uptime Monitor**
- **Heartbeat Monitor**

Dedicated uptime/heartbeat monitoring remains optional because established tools already cover that use case well. Gotify MU should integrate cleanly with those tools through Webhooks unless a clear need emerges for native monitoring.

### Not currently planned

From the current integration/plugin proposal, the following are intentionally not on the roadmap:

- Discord bridge
- Slack bridge
- Microsoft Teams bridge
- Prometheus Alertmanager bridge
- Grafana alerts plugin
- GitHub integration
- GitLab integration
- ntfy bridge
- Gotify-to-Gotify bridge
- SNMP trap receiver
- standalone message formatter plugin
- standalone rate-limit/deduplication plugin

### Plugin platform roadmap

Planned:

- Plugin Catalog
- first-party / third-party publisher identification
- custom catalog/repository URLs
- icons and metadata
- screenshots
- compatible Gotify MU version metadata
- server architecture/Go ABI compatibility checks
- checksums
- signatures/trust state
- one-click install
- update
- rollback
- uninstall
- automatic update policy
- plugin permission/capability presentation
- standardized event hooks for message ingest and optional integration extensions
- isolated plugin/service interface for plugins that should not execute inside the main Gotify MU process

## Search, archive, and retention

Planned:

- server-side full-text search
- date range filters
- Channel filters
- sender filters
- priority filters
- acknowledgement/resolution filters
- saved searches
- saved views
- message retention policies
- Channel-specific retention
- archive policies
- export
- bulk archive/delete controls

## Server administration and operations

Planned:

- richer system health dashboard
- database status
- storage usage
- message/attachment counts
- connected-client visibility
- WebSocket/session visibility
- plugin health
- queue/delivery diagnostics
- backup creation
- backup download
- restore workflow
- configuration export
- server update information
- safe update controls
- diagnostics bundle
- audit retention controls

## Gotify MU Android

A dedicated Android fork becomes necessary for interactive Gotify MU features while retaining compatibility with normal notification delivery.

Planned:

- Gotify MU branding/design system
- Channels
- Messages
- Archive
- full search
- Chat Channels
- compose
- replies/threads
- reactions
- acknowledgements
- resolve/reopen
- attachments
- per-device preferences
- quiet hours
- user profile/security
- MFA enrollment/elevation
- multiple-server support where practical

The Android fork should continue using the compatible Gotify REST/WebSocket behavior for existing features and layer Gotify MU endpoints on top.
