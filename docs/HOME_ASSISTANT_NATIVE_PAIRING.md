<p align="center">
  <img src="../assets/monita-banner.svg" alt="Monita" width="720">
</p>

# Monita for Home Assistant

Monita can connect with Home Assistant so automations can send notifications, images, and events to Monita Channels.

Two connection methods are supported:

- **Native pairing** with Monita for Home Assistant
- **Long-Lived Access Token** for direct Home Assistant connections

Native pairing is recommended for most new installations because it provides a guided setup without requiring the user to manually create and copy a Home Assistant token.

## What you can do

With Home Assistant connected to Monita, you can:

- send Home Assistant notifications to a Monita Channel
- include images such as doorbell or camera snapshots
- route different automations to different Channels
- use Home Assistant events as notification triggers
- send test events between Home Assistant and Monita
- keep notification history available in Monita

## Native pairing

In Monita:

1. Open **Integrations**.
2. Add or edit a **Home Assistant** connection.
3. Choose the native pairing option.
4. Generate a pairing code.

In Home Assistant:

1. Open the Monita integration.
2. Choose **Pair with Monita**.
3. Enter the pairing code.
4. Complete the setup flow.

Pairing codes are temporary and intended for one-time setup.

## Image notifications

Monita supports real image delivery from Home Assistant.

For example:

```text
Doorbell detects a person
        ↓
Home Assistant captures a camera snapshot
        ↓
Monita for Home Assistant sends the image to Monita
        ↓
The phone notification includes the image
        ↓
The same image remains available in Monita history
```

The image is transferred through the integration rather than relying on a local-only Home Assistant camera URL.

## Direct token connection

Long-Lived Access Token mode remains available for installations that prefer a direct Home Assistant connection.

Use native pairing unless you have a specific reason to manage the connection manually.

## Filtering and routing

A Home Assistant integration can be configured to route selected events and entities to a specific Monita Channel.

This makes it possible to create separate Channels for things such as:

- security alerts
- doorbells
- system warnings
- appliance notifications
- household reminders
- automation status

## Re-pairing

If the connection needs to be repaired, generate a new pairing code in Monita and complete the pairing flow again in Home Assistant.

Existing Monita Channels and notification history are not removed when a Home Assistant connection is re-paired.

## Troubleshooting

If pairing fails:

- confirm both systems can reach each other
- verify the server address
- generate a fresh pairing code
- confirm the Home Assistant integration is current
- check the Monita and Home Assistant logs for connection errors

## Security

Treat both Monita and Home Assistant as trusted services.

Use HTTPS for remote access and avoid exposing either administrative interface directly to the public internet.
