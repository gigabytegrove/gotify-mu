# Gotify MU Deployment Guide

This document is the maintained deployment, validation, update, backup, and rollback reference for Gotify MU.

## Release state

- **Current published baseline:** v0.2.2
- **Current preview candidate:** v0.5.0 on PR #22
- v0.5.0 is not the published baseline until the complete validation gate passes and the preview is accepted, merged, tagged, and released.

The release candidate must be validated at the exact commit that is merged. Documentation-only changes are part of the release candidate and must pass the same repository gate.

## Supported deployment models

Gotify MU supports:

1. Docker Compose with the Gotify MU updater helper.
2. Manual Docker deployment with a persistent data directory.
3. Native source builds for development/testing.

Docker is the recommended deployment model.

## Persistent data

The container stores persistent state under:

```text
/app/data
```

For the manual deployment used during project validation, that is mounted from:

```text
/opt/gotify-mu-data
```

Do not replace or delete the persistent data directory during a normal container upgrade.

A release that adds database migrations must have a verified pre-upgrade backup before the old container is replaced.

## Docker Compose deployment

Clone the repository:

```bash
cd /opt
git clone https://github.com/gigabytegrove/gotify-mu.git
cd gotify-mu
cp .env.example .env
```

Generate the private updater token:

```bash
openssl rand -hex 32
```

Edit `.env` and set at minimum:

```env
GOTIFY_MU_PORT=8080
GOTIFY_DEFAULTUSER_NAME=admin
GOTIFY_DEFAULTUSER_PASS=CHANGE-THIS-PASSWORD
GOTIFY_MU_UPDATER_TOKEN=CHANGE-THIS-TO-A-RANDOM-64-HEX-TOKEN
```

Build with the full server test suite enabled:

```bash
docker compose build --build-arg RUN_TESTS=1
```

Start the services:

```bash
docker compose up -d
```

Verify:

```bash
curl -fsS http://127.0.0.1:8080/health
docker ps --filter name=gotify-mu
docker logs --tail 100 gotify-mu
```

A healthy server returns:

```json
{"health":"green","database":"green"}
```

## Manual Docker deployment

A manual deployment should always keep application data outside the container.

Example build:

```bash
cd /opt/gotify-mu

COMMIT="$(git rev-parse HEAD)"
SHORT_COMMIT="$(git rev-parse --short HEAD)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

docker build \
  --build-arg BUILD_JS=1 \
  --build-arg RUN_TESTS=1 \
  --build-arg GO_VERSION=1.26.0 \
  --build-arg GOTIFY_MU_VERSION="preview-${SHORT_COMMIT}" \
  --build-arg GOTIFY_MU_COMMIT="${COMMIT}" \
  --build-arg GOTIFY_MU_BUILD_DATE="${BUILD_DATE}" \
  -f docker/Dockerfile \
  -t gotify-mu:preview \
  .
```

The build must finish successfully before the running server is stopped.

## Validation gate

Preview and development deployment follows a strict gate:

1. Build the complete Web UI.
2. Run the full Go test suite with `go test -v ./...`.
3. Build the final server binary/container.
4. Only after the complete build passes, stop the running container.
5. Create and verify a pre-upgrade data backup.
6. Preserve the previous container as a rollback container.
7. Start the new container against the persistent data directory.
8. Wait for both application and database health to become green.
9. Keep the previous container and pre-upgrade backup until live validation is complete.

The Dockerfile performs the full Go suite when built with:

```text
RUN_TESTS=1
```

A failed build or failed test suite must not proceed to the replacement stage.

## Safe backup before an upgrade

For a manual deployment using `/opt/gotify-mu-data`:

```bash
BACKUP_DIR="/opt/gotify-mu-backups"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

mkdir -p "${BACKUP_DIR}"

docker stop gotify-mu

tar \
  -C /opt \
  -czf "${BACKUP_DIR}/pre-upgrade-${TIMESTAMP}.tar.gz" \
  gotify-mu-data

tar -tzf "${BACKUP_DIR}/pre-upgrade-${TIMESTAMP}.tar.gz" >/dev/null
```

If backup creation or verification fails, restart the existing container and do not continue.

## Container replacement

Preserve the old container before starting the new one:

```bash
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
docker rename gotify-mu "gotify-mu-rollback-${TIMESTAMP}"
```

The replacement must use the same persistent data mount and required environment configuration.

Example:

```bash
docker run -d \
  --name gotify-mu \
  --restart unless-stopped \
  --network gotify-mu-system \
  -p 8080:80 \
  --env-file /path/to/preserved.env \
  -v /opt/gotify-mu-data:/app/data \
  gotify-mu:preview
```

Then verify:

```bash
curl -fsS http://127.0.0.1:8080/health
docker logs --tail 150 gotify-mu
```

## Rollback after a migrated preview

Do **not** simply start an older Gotify MU container against a database that has already been migrated by a newer preview.

For a rollback to the pre-upgrade version:

1. Stop and remove the failed/new container.
2. Preserve the failed preview data directory for investigation.
3. Restore the pre-upgrade data backup.
4. Rename the preserved old container back to `gotify-mu`.
5. Start the old container.
6. Verify health.

Example:

```bash
docker rm -f gotify-mu

mv /opt/gotify-mu-data "/opt/gotify-mu-data-failed-$(date +%Y%m%d-%H%M%S)"

tar -C /opt -xzf /opt/gotify-mu-backups/PRE-UPGRADE-BACKUP.tar.gz

docker rename OLD-ROLLBACK-CONTAINER gotify-mu
docker start gotify-mu

curl -fsS http://127.0.0.1:8080/health
```

Keep the failed preview data until the issue has been understood.

## Managed in-app updates

Published releases can be installed from **Settings → Software Update** when the updater helper is enabled.

The updater helper:

- runs without a published host port
- communicates with Gotify MU over the private Docker network
- requires a shared random token
- has access to the Docker socket so it can replace the application container
- preserves the current runtime configuration
- verifies the replacement
- automatically restores the previous container when replacement/startup verification fails

Because Docker socket access is privileged host access, only enable the helper on systems where managed updates are desired.

The shared token must never be committed to Git.

## Preview builds versus published releases

Preview branches are deployed manually and must pass the full validation gate.

The in-app updater is intended for **published numbered releases**. A preview branch should not be presented as a normal downloadable release until it has been accepted, merged, tagged, and published.

## v0.5.0 live validation checklist

Before v0.5.0 is locked as a release:

### Core compatibility

- existing administrator login works
- existing users, Groups, Channels, messages, archives, and tokens are intact
- existing application tokens continue to publish
- existing client-token access works
- official Gotify Android receive/display behavior remains normal
- shared and Global Channel delivery remains correct
- per-user mute, archive/restore, and destructive-action protections remain correct

### Identity and security

- local authentication works
- configured OIDC authentication remains functional
- LDAP / Active Directory authentication works when enabled
- TOTP MFA enrollment, verification, recovery codes, and policy enforcement work
- WebAuthn/passkey registration and authentication work
- active-session listing and revocation work
- service-account/API credentials enforce their configured scopes
- protected integration and connector secrets are not returned in plaintext
- login throttling and security-event auditing work

### Channels, permissions, and collaboration

- Channel roles enforce Owner, Manager, Publisher, Member, and Read Only permissions
- Group-to-Channel assignment grants the intended inherited access
- replies/threads, reactions, mentions, assignment, resolve/reopen, and read state work
- attachments can be created, retrieved, and removed under the correct permissions
- message templates and saved searches persist and execute correctly
- acknowledgement history is visible and accurate

### Native integrations

- Webhook Router accepts valid requests and rejects invalid signature, replay, CIDR, rate-limit, and oversized-payload cases
- MQTT connects using configured protocol/TLS settings and routes QoS 0/1/2 messages
- Home Assistant receives filtered events and can send an outbound test event
- integration health, last-event/message, reconnect, and error state are accurate

### First-party connectors

- Email Delivery sends qualifying notifications through SMTP
- SMTP Receiver accepts permitted mail and routes it to the intended Channel
- RSS / Atom polling applies duplicate, title, and category filters
- Syslog Receiver applies source/facility/severity filters and duplicate suppression
- Calendar / iCal polling generates reminders once per intended event/window

### Automation

- one-time, hourly, daily, weekly, and cron Scheduled Notifications run correctly
- excluded dates, end dates, maximum runs, and misfire behavior are honored
- schedule execution history records outcomes
- Quiet Hours suppression and deferred delivery work
- Digest queues and stored summaries work
- Escalations honor acknowledgement cancellation, repeat rules, targets, and depth protection
- multi-instance leases prevent duplicate scheduler/integration execution

### Plugins

- verified plugin upload/install works
- checksum/signature/trusted-key policy is enforced
- Plugin Catalog install/update works
- plugin update and uninstall work
- plugin-created notifications pass through Gotify MU delivery policy

### Operations and updater

- Operations counters and diagnostics load
- backup creation/download works
- restore staging validates an accepted backup without modifying live data until explicitly applied
- configuration and Audit Log export work
- retention cleanup runs without removing protected/live records
- Settings remains stable and does not enter a refresh loop
- managed update status exposes meaningful progress
- checksum verification, full-test build, health verification, and automatic container rollback behave correctly

### Final deployment gate

- complete Web UI build passes
- Go lint passes
- complete Go test suite passes
- repository consistency checks pass
- production Docker build with `RUN_TESTS=1` passes
- filesystem/dependency and built-container vulnerability gates pass
- SPDX SBOM is generated
- database migrations succeed against a verified copy of the existing deployment data
- application and database health are green after upgrade
- rollback container and verified pre-upgrade backup are retained until live acceptance is complete

## Release publication

After a preview is accepted:

1. Merge the release PR.
2. Confirm `VERSION` contains the intended release number.
3. Create/publish the release tag.
4. Publish release notes and downloadable assets.
5. Install the official numbered release build.
6. Verify health and core compatibility again.
7. Only then remove the previous rollback container and old backup according to the administrator's retention policy.

## Secrets

Never commit:

- administrator passwords
- updater tokens
- MQTT passwords
- Home Assistant long-lived access tokens
- SMTP/mail credentials
- other integration credentials

Saved integration credentials must remain masked in the Web UI and should not be written into ordinary application logs.
