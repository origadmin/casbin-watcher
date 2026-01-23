# Redis Streams Driver for Casbin Watcher

This directory contains the Redis Streams (`redisstream`) driver for `casbin-watcher`. This driver uses Redis Streams to
broadcast policy updates, providing a persistent and scalable messaging solution.

## How it Works

The `redisstream` driver leverages Watermill's `redisstream` Pub/Sub implementation.

- **Publish**: Policy updates are published to a Redis Stream topic.
- **Subscribe**: A subscriber listens to the stream as part of a consumer group. Each watcher instance should ideally
  have a unique consumer group to ensure it receives all updates. If a consumer group is not specified, a unique one
  will be generated automatically.

This driver is suitable for production environments where reliable and persistent messaging is required.

## Configuration

The driver is configured using a URL.

### URL Format

```
redisstream://user:password@host:port/db_number?pool_size=10&consumer_group=my_group
```

- **Connection**: Standard Redis connection details (`user`, `password`, `host`, `port`).
- **Database**: The Redis database number, provided in the `path` part of the URL. Defaults to `0`.
- **Parameters**: Additional options are configured via query parameters.

### Configuration Parameters

| Parameter        | Type      | Default                 | Description                                                                                                                                      | Example                       |
|------------------|-----------|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------|
| `pool_size`      | `integer` | (Redis default)         | The connection pool size for the Redis client.                                                                                                   | `pool_size=20`                |
| `consumer_group` | `string`  | `casbin-watcher-<UUID>` | The name of the Redis Streams consumer group. If not provided, a unique name is generated to ensure the watcher acts as an independent consumer. | `consumer_group=casbin_nodes` |

## Usage Example

### Basic Connection

```
redisstream://localhost:6379/1
```

### With Consumer Group

```
redisstream://localhost:6379/1?consumer_group=my_app_casbin_watcher
```
