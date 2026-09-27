<p align="center">
  <img src="../assets/monita-banner.svg" alt="Monita" width="720">
</p>

# Deploying Monita

This guide covers a normal Monita installation, updates, backups, and basic troubleshooting.

## Recommended installation

Docker Compose is the recommended way to run Monita. The standard deployment downloads the published Monita image from GitHub Container Registry, so the server does not need to compile the Web UI or Go application.

### Requirements

You need:

- a Linux system
- Git
- Docker
- Docker Compose

### Install

```bash
cd /opt
git clone https://github.com/gigabytegrove/monita.git
cd monita
cp .env.example .env
nano .env
```

Set at least:

```env
GOTIFY_DEFAULTUSER_NAME=admin
GOTIFY_DEFAULTUSER_PASS=CHANGE-THIS-PASSWORD
```

Then start Monita:

```bash
docker compose pull
docker compose up -d
```

Open:

```text
http://SERVER-IP:8080
```

## Prebuilt release image archives

If your environment cannot pull directly from GitHub Container Registry, Monita also publishes prebuilt Docker image archives with each supported release.

Download the archive that matches your server architecture, verify it with the published checksum file, load it with Docker, and start Monita with Docker Compose. This path does not compile Monita locally.

Supported release archives:

- `linux-amd64` for standard 64-bit Intel/AMD systems
- `linux-arm64` for 64-bit ARM systems

## Configuration

Monita uses environment variables from `.env`.

Common settings include:

```env
MONITA_PORT=8080
MONITA_DATA_DIR=./data
MONITA_RECEIVER_BIND=127.0.0.1
MONITA_SMTP_PORT=2525
MONITA_SYSLOG_PORT=5514
```

Gotify-compatible `GOTIFY_*` settings remain supported where required for compatibility.

## Persistent data

Monita stores persistent application data in the configured data directory.

For the default Compose setup:

```text
./data
```

Keep this directory when rebuilding, moving, or updating Monita.

## Updating

Normal updates use published Monita images. Building from source is not part of the standard update path.

### In the Web UI

Go to:

**Settings → Software Update**

Monita checks for published releases and can install supported updates from the Web UI.

### Manual update

```bash
cd /opt/monita
git pull --ff-only origin master
docker compose pull
docker compose up -d
```

After an update, verify that the container is healthy:

```bash
docker ps --filter name=monita
curl -fsS http://127.0.0.1:8080/health
```

## Backups

Before major upgrades or configuration changes, back up the Monita data directory.

For the default installation:

```bash
cd /opt/monita
tar -czf monita-backup-$(date +%Y%m%d-%H%M%S).tar.gz data
```

Store important backups somewhere outside the Monita server.

If your installation uses external databases, custom certificates, or external authentication, include the related configuration in your backup plan.

## Restoring

Stop Monita before replacing its persistent data.

```bash
cd /opt/monita
docker compose down
```

Restore your saved data, then start Monita again:

```bash
docker compose up -d
```

## Logs

View recent logs:

```bash
docker logs --tail 200 monita
```

Follow logs live:

```bash
docker logs -f monita
```

## Health check

```bash
curl -fsS http://127.0.0.1:8080/health
```

A healthy installation should report a healthy application and database.

## Ports

The default Web UI/API port is:

```text
8080
```

Optional receiver services may use additional ports depending on your configuration.

Only expose ports that your environment actually needs.

## HTTPS

For internet-facing deployments, place Monita behind a trusted HTTPS reverse proxy.

Do not expose administrative interfaces or optional receiver ports publicly unless you specifically need them and have secured them appropriately.

## Troubleshooting

### Container is not running

```bash
docker ps -a --filter name=monita
docker logs --tail 200 monita
```

### Web UI does not load

Confirm the configured port and verify the health endpoint.

### Update fails

Check the Software Update page and container logs. Your existing persistent data should remain unchanged if the application container is rebuilt.

### Data appears missing

Confirm that the same persistent data directory is still mounted into the container.

## Local source builds

Normal installations should use the published container image. Building from source is intended for development or troubleshooting.

To build locally with Compose:

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

## Manual Docker deployment

Compose is recommended, but Monita can also be run directly with Docker.

```bash
docker build   --build-arg BUILD_JS=1   --build-arg GO_VERSION=1.26.0   -f docker/Dockerfile   -t monita:local   .
```

Then run the image with a persistent `/app/data` mount and the environment variables required by your installation.

## Security

For deployment security guidance, see [SECURITY.md](../SECURITY.md) and [Security overview](SECURITY_ROADMAP.md).
