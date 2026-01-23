# RabbitMQ (AMQP) Driver for Casbin Watcher

This directory contains the RabbitMQ driver for `casbin-watcher`. This driver uses the AMQP protocol to communicate with
a RabbitMQ broker, providing a robust and widely-used messaging backend for broadcasting policy updates.

## How it Works

The `rabbitmq` driver is built on Watermill's `amqp` Pub/Sub implementation.

- **Publish/Subscribe**: It uses a durable topic exchange to publish messages. Each subscriber listens to a durable
  queue that is bound to this exchange. This setup ensures that messages are persisted even if the broker restarts and
  that multiple watchers can receive the same policy update.
- **Configuration**: The driver is configured using a standard AMQP connection URI.

This driver is well-suited for production environments that require reliable, persistent, and scalable messaging.

## Configuration

The driver is configured using a URL.

### URL Format

The driver uses a standard AMQP 0-9-1 connection URI.

```
amqp://user:password@host:port/vhost
```

### Configuration Parameters

The configuration is entirely derived from the AMQP connection URI itself.

| ? Part          | Description                                                                | Example           |
|-----------------|----------------------------------------------------------------------------|-------------------|
| `scheme`        | The protocol, which must be `amqp`.                                        | `amqp://...`      |
| `user:password` | (Optional) Authentication credentials for the RabbitMQ broker.             | `guest:guest@...` |
| `host:port`     | The address and port of the RabbitMQ server.                               | `localhost:5672`  |
| `vhost`         | (Optional) The virtual host to use. If omitted, the default (`/`) is used. | `.../my_vhost`    |

## Usage Example

### Basic Connection

```
amqp://guest:guest@localhost:5672/
```

### With Custom Virtual Host

```
amqp://guest:guest@localhost:5672/my_vhost
```
