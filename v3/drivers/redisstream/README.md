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

### Usage Example

```go
import (
    "context"
    "log"

    "github.com/casbin/casbin/v2"
    "github.com/origadmin/casbin-watcher/v3"
    _ "github.com/origadmin/casbin-watcher/v3/drivers/redisstream" // Register the driver
)

func main() {
    // Connect to a Redis server on DB 1, with a specific consumer group.
    // The topic for policy updates is "casbin_updates".
    connectionURL := "redisstream://localhost:6379/1?consumer_group=my_app_casbin_watcher"
    
    w, err := watcher.NewWatcher(context.Background(), connectionURL, "casbin_updates")
    if err != nil {
        log.Fatalf("Failed to create watcher: %v", err)
    }

    e, err := casbin.NewEnforcer("model.conf", "policy.csv")
    if err != nil {
        log.Fatalf("Failed to create enforcer: %v", err)
    }

    err = e.SetWatcher(w)
    if err != nil {
        log.Fatalf("Failed to set watcher: %v", err)
    }
    
    // When you call e.SavePolicy(), the update will be sent via Redis Streams.
}
```
