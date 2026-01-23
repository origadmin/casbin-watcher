# NATS Driver for Casbin Watcher

This directory contains the NATS driver for `casbin-watcher`. It supports both core NATS and NATS JetStream
functionalities, configurable via the connection URL.

## Features

- **Core NATS & JetStream**: Seamlessly works with both modes.
- **High Customization**: Most NATS and JetStream features can be configured via URL parameters.
- **Robust Connection**: Built-in connection retry logic.
- **Graceful Shutdown**: Configurable timeout for closing the subscriber.
- **Flexible Messaging**: Supports different message delivery policies and serialization formats.
- **Flexible Provisioning**: Separates JetStream activation from stream auto-provisioning for production environments.

## Configuration

The driver is configured using a URL. The general format is:

```
nats://<host>:<port>/<topic>?<parameters>
```

- `<host>:<port>`: The address of your NATS server.
- `<topic>`: The base topic used for Casbin policy updates.
- `<parameters>`: Query parameters to customize the driver's behavior.

### JetStream Mode

To enable JetStream, you **must** include `jetstream=true` in the URL parameters.

By default, when JetStream is enabled, the driver will also attempt to automatically create a stream with the **same
name as the topic** if it doesn't exist (`auto_provision=true`). This is convenient for development but may not be
desirable in production.

For production environments where streams are managed manually, you can disable this behavior by setting
`auto_provision=false`.

### Configuration Parameters

| Parameter                 | Type       | Default  | Description                                                                                                                                             | Example                         |
|---------------------------|------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------|
| `jetstream`               | `boolean`  | `false`  | Set to `true` to enable NATS JetStream mode.                                                                                                            | `jetstream=true`                |
| `auto_provision`          | `boolean`  | `true`   | **(JetStream only)** If `true`, automatically creates the stream if it doesn't exist. Set to `false` in environments where streams are pre-provisioned. | `auto_provision=false`          |
| `queue_group`             | `string`   | `""`     | The queue group name. In JetStream mode, this is used as the **Durable Name** for the consumer, making subscriptions persistent.                        | `queue_group=my-app`            |
| `marshaler`               | `string`   | `gob`    | The serialization format for messages. Can be `gob` or `json`.                                                                                          | `marshaler=json`                |
| `connect_timeout`         | `duration` | `(none)` | Connection timeout for the NATS client.                                                                                                                 | `connect_timeout=10s`           |
| `reconnect_wait`          | `duration` | `(none)` | The time to wait between reconnection attempts.                                                                                                         | `reconnect_wait=5s`             |
| `close_timeout`           | `duration` | `30s`    | The time to wait for the subscriber to gracefully close, allowing in-flight messages to be processed.                                                   | `close_timeout=1m`              |
| `retry_on_failed_connect` | `boolean`  | `true`   | If `true`, the client will attempt to reconnect if the initial connection fails.                                                                        | `retry_on_failed_connect=false` |
| `deliver_policy`          | `string`   | `all`    | **(JetStream only)** Defines where to start receiving messages. Can be `all`, `last`, `new`, or `last_per_subject`.                                     | `deliver_policy=new`            |
| `ack_async`               | `boolean`  | `false`  | **(JetStream only)** If `true`, enables asynchronous message acknowledgements for higher performance.                                                   | `ack_async=true`                |
| `track_msg_id`            | `boolean`  | `false`  | **(JetStream only)** If `true`, uses the Watermill message UUID as the `Nats-Msg-Id` header for more robust deduplication.                              | `track_msg_id=true`             |

---

## Usage Examples

### Basic Core NATS

Connects to a standard NATS server.

```
nats://localhost:4222/casbin-updates
```

### Basic JetStream (with Auto Provisioning)

Connects to NATS JetStream, automatically creating a stream named `casbin-updates` and a durable consumer for the
`my-app` group. This is ideal for development.

```
nats://localhost:4222/casbin-updates?jetstream=true&queue_group=my-app
```

### JetStream for Production (Manual Provisioning)

Connects to a pre-existing JetStream stream named `casbin-events` without trying to create it. This is the recommended
approach for production.

```
nats://localhost:4222/casbin-events?jetstream=true&auto_provision=false&queue_group=my-app-v1
```

### Highly Customized JetStream

Connects to JetStream with several advanced options:

- Disables auto-provisioning.
- Uses `json` for messages.
- Subscribes to only new messages (`deliver_policy=new`).
- Enables asynchronous acknowledgements.
- Sets a 10-second connection timeout and a 1-minute graceful shutdown timeout.

```
nats://localhost:4222/casbin-events?jetstream=true&auto_provision=false&queue_group=my-app-v1&marshaler=json&deliver_policy=new&ack_async=true&connect_timeout=10s&close_timeout=1m
```
