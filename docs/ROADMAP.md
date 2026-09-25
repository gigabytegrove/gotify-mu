# Gotify MU Product Roadmap

Gotify MU keeps Gotify REST/WebSocket and token compatibility where practical while extending the server into a multi-user notification and operations platform.

## v0.5 platform

The following capabilities are implemented in the v0.5 preview branch.

### Identity and security

- local username/password authentication
- OIDC
- LDAP / Active Directory
- TOTP MFA
- recovery codes
- WebAuthn/passkeys
- administrator MFA policy
- configurable session/elevation policy
- login throttling
- active-session visibility and revocation
- scoped service accounts/API credentials
- encrypted server-side secret storage
- sensitive request/log redaction
- security/admin Audit Log with export and retention

### Channels and permissions

- shared Channels
- Global Channels
- ownership transfer
- Owner / Manager / Publisher / Member / Read Only roles
- Group-to-Channel assignment
- per-user notification preference
- per-user archive/restore
- Chat Channel posting
- safe destructive-action rules

### Collaboration

- replies and threads
- reactions
- mentions
- message acknowledgement and acknowledgement history
- assignment
- resolve/reopen
- read/unread state
- attachments
- message templates
- saved searches

### Notification intelligence

- Quiet Hours suppression
- deferred Quiet Hours delivery
- priority bypass
- Digests
- stored Digest summaries
- one-time/hourly/daily/weekly schedules
- cron schedules
- timezone-aware scheduling
- excluded dates
- end dates and maximum runs
- misfire policy
- schedule run history
- Channel/user/Group Escalation targets
- repeat Escalations
- acknowledgement cancellation
- structured escalation lineage
- multi-instance scheduler leases and delivery deduplication

### Native integrations

#### Webhook Router

- generated inbound URLs
- encrypted secrets and hashed lookup
- HMAC request signing
- replay protection
- source CIDR restrictions
- per-route/per-source rate limiting
- JSON title/message/priority extraction
- array-aware field paths
- conditional field/value matching
- title/message templates
- explicit request-size limit
- retained delivery history

#### MQTT

- MQTT 5
- MQTT 3.1.1
- QoS 0/1/2 receive flows
- bounded packet size
- username/password authentication
- custom CA certificates
- mutual TLS client certificates
- topic subscriptions
- reconnect behavior
- connection testing
- status/last-connect/last-message/error visibility

#### Home Assistant

- WebSocket event subscriptions
- event-type filtering
- entity filtering
- event-data field/value filtering
- Channel routing
- outbound events
- test action
- reconnect and health/error visibility

### First-party connectors

- Email Delivery
- SMTP Receiver
- RSS / Atom Monitor
- Syslog Receiver
- Calendar / iCal

These are native first-party connectors rather than optional third-party plugins because they are supported as part of the Gotify MU server.

### Plugin platform

- existing compatible Gotify Go plugin support
- administrator Web UI install
- SHA-256 verification
- Ed25519 signatures and trusted signing keys
- unsigned installation disabled by default
- Plugin Catalog
- catalog install/update
- verified manual update staging
- uninstall
- plugin-created messages routed through Gotify MU delivery policy

Native Go plugins remain trusted in-process extensions. They are not a sandbox boundary.

### Operations

- system/operations dashboard
- database and storage information
- queue/automation counters
- active sessions
- backup creation/download
- restore staging
- configuration export
- diagnostics
- audit export/retention
- message/attachment/automation/connector cleanup policies
- managed in-app Docker updates with checksum verification, full-test build gate, health verification, and rollback

### Release engineering

- read-only pull-request CI
- SHA-pinned GitHub Actions
- Web UI build
- Go lint
- full Go tests
- repository checks
- production Docker build with tests enabled
- dependency/filesystem vulnerability scanning
- container vulnerability scanning
- SPDX SBOM
- release SHA-256 checksums
- build provenance attestation

## After v0.5

The v0.5 goal is to close the server-side audit backlog rather than continuously add unrelated scope. Future work should be driven by real operational feedback.

Potential later work:

- an isolated out-of-process extension protocol for integrations that should not execute as trusted native Go plugins
- more advanced outbound integration workflow composition if real use cases require it
- larger-scale performance tuning based on measured installations
- additional first-party connectors only when there is a demonstrated need

## Gotify MU Android

A dedicated Android client remains a separate client project. The server preserves normal official Gotify Android receive/display compatibility.

A Gotify MU Android client can later expose MU-specific features such as:

- Channel management
- compose/send
- threads/replies
- reactions
- acknowledgements
- assignment and resolve/reopen
- attachments
- Quiet Hours/Digest controls
- account security/MFA
- multiple-server support

The Android client should continue using compatible Gotify REST/WebSocket behavior for existing functions and layer Gotify MU endpoints on top.
