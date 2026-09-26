# Home Assistant native pairing contract

Gotify MU supports two Home Assistant connection methods:

1. Long-Lived Access Token (existing direct Home Assistant API/WebSocket connection).
2. Native pairing with the `gigabytegrove/gotify-mu-ha` custom integration.

The native path is intended to remove the requirement for users to create a Home Assistant Long-Lived Access Token.

## Gotify MU flow

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

Gotify MU returns a one-time pairing code in the form:

```text
<integration-id>.<random-secret>
```

The code expires after 15 minutes and is stored only as a SHA-256 hash.

A new code can be requested by an authenticated administrator:

```text
POST /integration/home-assistant/:id/pairing
```

## Home Assistant pairing request

The `gotify-mu-ha` integration already knows the Gotify MU server URL from its config entry.

It should expose an options/config flow named **Pair with Gotify MU server** that asks for:

- pairing code
- Home Assistant URL reachable by Gotify MU, when it cannot be determined automatically

The HA integration should register a private webhook handler with a random webhook ID and then call:

```text
POST <gotify-mu-server>/integrations/home-assistant/native/pair
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

## HA -> Gotify MU events

When paired, the HA integration sends selected Home Assistant events to:

```text
POST <gotify-mu-server><eventPath>
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

Gotify MU applies the configured event type, entity ID, data-field and data-value filters before routing the event into the selected Channel.

## Gotify MU -> Home Assistant events

Gotify MU posts to the webhook URL supplied during pairing:

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
    "message": "Gotify MU connection test"
  }
}
```

The HA webhook handler must compare the Bearer credential to the stored shared secret and reject invalid requests. On success, it should fire the requested Home Assistant event on the HA event bus.

## Required HA-side behavior

- Keep the existing application-token/client-token Gotify MU notification functionality unchanged.
- Add native server pairing as an optional capability; do not replace the current setup flow.
- Use Home Assistant config-entry storage for bridge credentials.
- Redact the shared secret and pairing data from diagnostics/logs.
- Use a random HA webhook ID.
- Do not accept the pairing code after a successful pair.
- Do not execute arbitrary Home Assistant services from bridge payloads.
- Native inbound bridge payloads may fire Home Assistant events only.
- Preserve LLT mode in Gotify MU as a fully supported fallback.
