<p align="center">
  <img src="../assets/monita-banner.png" alt="Monita" width="720">
</p>

# Home Assistant native pairing contract

Monita supports two Home Assistant connection methods:

1. Long-Lived Access Token (existing direct Home Assistant API/WebSocket connection).
2. Native pairing with the `gigabytegrove/monita-ha` custom integration.

The native path is intended to remove the requirement for users to create a Home Assistant Long-Lived Access Token.

## Monita flow

An administrator creates a Home Assistant connection with:

```json
{
  "name": "Home Assistant",
  "applicationId": 1,
  "connectionMode": "integration",
  "eventType": "",
  "entityIds": "",
  "dataField": "",
  "dataValue": "",
  "enabled": true
}
```

Monita returns a one-time pairing code in the form:

```text
<integration-id>.<random-secret>
```

The code expires after 15 minutes and is stored only as a SHA-256 hash.

A new code can be requested by an authenticated administrator:

```text
POST /integration/home-assistant/:id/pairing
```

## Home Assistant pairing request

The `monita-ha` integration already knows the Monita server URL from its config entry.

It should expose an options/config flow named **Pair with Monita server** that asks for:

- pairing code
- Home Assistant URL reachable by Monita when it cannot be determined automatically, with an optional manual override when the automatically selected URL is not reachable from the Monita server

The HA integration should register a private webhook handler with a random webhook ID and then call:

```text
POST <monita-server>/integrations/home-assistant/native/pair
Content-Type: application/json
```

Body:

```json
{
  "pairingCode": "<integration-id>.<random-secret>",
  "webhookUrl": "https://home-assistant.example/api/webhook/<random-webhook-id>"
}
```

Successful response:

```json
{
  "integrationId": 1,
  "secret": "<shared-native-bridge-secret>",
  "eventPath": "/integrations/home-assistant/native/1/event"
}
```

The Home Assistant integration must store `secret`, `integrationId`, and `eventPath` in config-entry storage and redact them from diagnostics.

The pairing code is one-time use. A successful pairing invalidates it immediately.

## HA -> Monita events

When paired, the HA integration sends selected Home Assistant events to:

```text
POST <monita-server><eventPath>
Authorization: Bearer <shared-native-bridge-secret>
Content-Type: application/json
```

Body:

```json
{
  "eventType": "state_changed",
  "data": {
    "entity_id": "binary_sensor.front_door"
  }
}
```

Monita applies the configured event type, entity ID, data-field and data-value filters before routing the event into the selected Channel.

## Monita -> Home Assistant events

Monita posts to the webhook URL supplied during pairing:

```text
POST <home-assistant-webhook-url>
Authorization: Bearer <shared-native-bridge-secret>
Content-Type: application/json
```

Body:

```json
{
  "eventType": "gotify_mu_test",
  "data": {
    "message": "Monita connection test"
  }
}
```

The HA webhook handler must compare the Bearer credential to the stored shared secret and reject invalid requests. On success, it should fire the requested Home Assistant event on the HA event bus.

## Required HA-side behavior

- Keep the existing application-token/client-token Monita notification functionality unchanged.
- Add native server pairing as an optional capability; do not replace the current setup flow.
- Use Home Assistant config-entry storage for bridge credentials.
- Redact the shared secret and pairing data from diagnostics/logs.
- Use a random HA webhook ID.
- Do not accept the pairing code after a successful pair.
- Do not execute arbitrary Home Assistant services from bridge payloads.
- Native inbound bridge payloads may fire Home Assistant events only.
- Preserve LLT mode in Monita as a fully supported fallback.


## Native unpair / revoke

Home Assistant removes native pairing by revoking the shared bridge credential on Monita before deleting its local copy:

```text
DELETE <monita-server>/integrations/home-assistant/native/<integration-id>
Authorization: Bearer <shared-native-bridge-secret>
```

A successful revoke returns HTTP 204. Monita clears the stored native webhook URL, shared secret, and any outstanding pairing state while preserving the normal Home Assistant connection record and its filters.

Home Assistant must not silently discard local credentials when the revoke request fails. A force-local-remove escape hatch may be offered for recovery when the Monita server is unavailable or the remote connection has already been replaced.

Regenerating a pairing code for an already paired connection does not tear down the working native bridge. The existing webhook and shared secret remain valid until a replacement pairing succeeds. The Monita admin UI exposes **Generate Repair Code** for an already paired native connection.

## Health and delivery expectations

- HTTP 400, 401, or 404 from the pairing endpoint indicates an invalid or no-longer-valid pairing code; HTTP 410 indicates expiration.
- A paired client should expose bridge health separately from simple credential presence.
- HTTP 401/403 from the native event endpoint means the shared credential is no longer valid and should surface as repair required.
- Transient event-delivery failures should use bounded retry/backoff rather than silently dropping the first failed event.
