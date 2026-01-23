# etcd Driver for Casbin Watcher

This directory contains the etcd driver for `casbin-watcher`.

**Important:** This driver is a **manual implementation** that uses etcd's `Put` and `Watch` APIs to simulate a Pub/Sub
system. It is **not** based on an official `watermill-etcd` library and has significant limitations compared to a
full-featured message broker.

## How it Works

- **Publish**: A `Publish` operation performs an `etcdctl put <topic>/<uuid> <payload>` command.
- **Subscribe**: A `Subscribe` operation creates a `watch` on the specified topic prefix.

This makes it suitable for state synchronization tasks, like broadcasting policy updates, but not for general-purpose
message queuing.

## Limitations & Delivery Semantics

- **At-Most-Once Delivery**: This driver provides **at-most-once** delivery semantics. If a consumer crashes after
  receiving a message but before processing it, the message is **permanently lost**.
- **No Ack/Nack Mechanism**: The `Message.Ack()` and `Message.Nack()` calls are no-ops. The driver does not support
  message redelivery on failure.
- **No Message Persistence for Consumers**: The watcher only receives updates that occur while it is running. It will
  not receive messages that were published while it was offline.

Due to these limitations, this driver is recommended for use cases where the occasional loss of an update message is not
critical, as Casbin policies can be periodically reloaded from the source.

## Configuration

The driver is configured using a URL.

### URL Format

```
etcd://user:password@host1:port1,host2:port2/topic?dial_timeout=5s
```

- **Endpoints**: A comma-separated list of etcd server endpoints, provided in the `host` part of the URL.
- **Topic**: The topic to subscribe to, provided in the `path` part of the URL.
- **Authentication**: Username and password can be provided in the user info part of the URL.
- **Parameters**: Additional options are configured via query parameters.

### Configuration Parameters

| Parameter      | Type       | Default | Description                                    | Example            |
|----------------|------------|---------|------------------------------------------------|--------------------|
| `dial_timeout` | `duration` | `5s`    | Timeout for establishing a connection to etcd. | `dial_timeout=10s` |

### Usage Example

```go
import (
"context"
"log"

"github.com/casbin/casbin/v2"
"github.com/origadmin/casbin-watcher/v3"
_ "github.com/origadmin/casbin-watcher/v3/drivers/etcd" // Register the driver
)

func main() {
// Connect to a single etcd server with a 10-second timeout.
// The topic for policy updates is "casbin_policy".
connectionURL := "etcd://127.0.0.1:2379/casbin_policy?dial_timeout=10s"

w, err := watcher.NewWatcher(context.Background(), connectionURL)
if err != nil {
log.Fatalf("Failed to create watcher: %v", err)
}

// Connect to a cluster with authentication.
// connectionURL = "etcd://myuser:mypass@node1:2379,node2:2379,node3:2379/casbin_policy"
// w, err := watcher.NewWatcher(context.Background(), connectionURL)

e, err := casbin.NewEnforcer("model.conf", "policy.csv")
if err != nil {
log.Fatalf("Failed to create enforcer: %v", err)
}

err = e.SetWatcher(w)
if err != nil {
log.Fatalf("Failed to set watcher: %v", err)
}

// ...
}
```
