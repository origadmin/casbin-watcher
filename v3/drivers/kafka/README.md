# Kafka Driver for Casbin Watcher

This directory contains the Kafka driver for `casbin-watcher`. It is based on the `watermill-kafka/v3` library and uses
`github.com/IBM/sarama` as the underlying Kafka client.

## How it Works

This driver uses Kafka for broadcasting policy updates.

- **Publisher**: Sends messages to a specified Kafka topic.
- **Subscriber**: Receives messages from a Kafka topic using a consumer group.

## Authentication

The driver currently supports SASL/PLAIN authentication. To use it, you need to provide the following URL parameters:

- `sasl_enable=true`
- `sasl_user=<your_username>`
- `sasl_password=<your_password>`

Support for other SASL mechanisms (like SCRAM) can be added in the future if needed.

## Configuration

The driver is configured using a URL.

### URL Format

```
kafka://<broker1:port1>,<broker2:port2>?<parameters>
```

- `<broker1:port1>,<broker2:port2>`: A comma-separated list of Kafka brokers.
- The topic for publishing and subscribing is passed from the `watcher.NewWatcher` call, not from the URL.

### Configuration Parameters

| Parameter        | Type      | Default          | Description                                                                                                                                                       | Example                            |
|------------------|-----------|------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------|
| `consumer_group` | `string`  | `casbin-watcher` | The consumer group to use. All watchers with the same consumer group will form a group, and messages will be distributed among them (competing consumer pattern). | `consumer_group=my-app-group`      |
| `marshaler`      | `string`  | `default`        | The serialization format for messages. Can be `default` (raw `[]byte`) or `json`.                                                                                 | `marshaler=json`                   |
| `initial_offset` | `string`  | `newest`         | The initial offset to start consuming from. Can be `newest` (start from the latest message) or `oldest` (start from the very beginning).                          | `initial_offset=oldest`            |
| `sasl_enable`    | `boolean` | `false`          | Set to `true` to enable SASL/PLAIN authentication.                                                                                                                | `sasl_enable=true`                 |
| `sasl_user`      | `string`  | `(none)`         | The SASL username. Required if `sasl_enable` is `true`.                                                                                                           | `sasl_user=my-user`                |
| `sasl_password`  | `string`  | `(none)`         | The SASL password. Required if `sasl_enable` is `true`.                                                                                                           | `sasl_password=my-secret-password` |

## Usage Example

### Basic Connection

```
kafka://localhost:9092
```

### With SASL Authentication

```
kafka://kafka-broker1:9092?consumer_group=my-app&sasl_enable=true&sasl_user=my-user&sasl_password=my-password
```

### With Custom Marshaler

```
kafka://localhost:9092?consumer_group=my-app&marshaler=json
```
