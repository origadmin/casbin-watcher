# Casbin Watermill Watcher

[![Go Report Card](https://goreportcard.com/badge/github.com/casbin/watermill-adapter)](https://goreportcard.com/report/github.com/casbin/watermill-adapter)
[![Build Status](https://github.com/casbin/watermill-adapter/workflows/build/badge.svg?branch=master)](https://github.com/casbin/watermill-adapter/actions)
[![Godoc](https://godoc.org/github.com/casbin/watermill-adapter?status.svg)](https://godoc.org/github.com/casbin/watermill-adapter)

This is a [Casbin](https://casbin.org/) watcher based on [Watermill](https://watermill.io/), a Go library for working
efficiently with message streams. With Watermill's support for various Pub/Sub backends, this watcher allows for
flexible integration with numerous messaging systems.

## Installation

```bash
go get github.com/origadmin/casbin-watcher/v3
```

## Usage

To use the watcher, you need to create a new watcher instance with a connection URL that specifies the driver and its
configuration.

```go
import (
"context"
"log"

"github.com/casbin/casbin/v3"
"github.com/origadmin/casbin-watcher/v3"
// Import the specific driver you want to use
_ "github.com/origadmin/casbin-watcher/v3/drivers/redis"
)

func main() {
// The connection URL for the desired driver.
// See "Driver Configuration" section for details.
connectionURL := "redis://localhost:6379/0"

// Create a new watcher.
w, err := watcher.NewWatcher(context.Background(), connectionURL)
if err != nil {
log.Fatalf("Failed to create watcher: %v", err)
}

// Initialize the enforcer.
e, err := casbin.NewEnforcer("model.conf", "policy.csv")
if err != nil {
log.Fatalf("Failed to create enforcer: %v", err)
}

// Set the watcher for the enforcer.
if err := e.SetWatcher(w); err != nil {
log.Fatalf("Failed to set watcher: %v", err)
}

// The watcher will now automatically update other enforcer instances
// when the policy changes.
// For example, after e.SavePolicy() or e.AddPolicy(), etc.
}
```

## Driver Configuration

The `connectionURL` format is `scheme://<host>/<path>?<query_params>`. The scheme determines which driver is used.

### In-Memory (`mem`)

Ideal for testing and development.

- **Scheme:** `mem://`
- **Format:** `mem://<topic>?shared=<true|false>`
- **Example:** `mem://casbin-updates?shared=true`
- **Parameters:**
    - `shared`: (Optional) `true` (default) makes all watchers share the same in-memory pub/sub instance. `false`
      creates a new, isolated instance for each watcher.

### NATS (`nats`)

- **Scheme:** `nats://`
- **Format:** `nats://<host>:<port>/<topic>?channel=<channel_name>[&jetstream=true]`
- **Example:** `nats://localhost:4222/casbin-updates?channel=my-channel&jetstream=true`
- **Parameters:**
    - `channel`: (Required) The NSQ channel name.
    - `jetstream`: (Optional) Set to `true` to enable JetStream support.

### Redis (`redis`)

Uses Redis Streams.

- **Scheme:** `redis://`
- **Format:** `redis://<user>:<password>@<host>:<port>/<db_number>?pool_size=<size>`
- **Example:** `redis://:mypassword@localhost:6379/0?pool_size=10`
- **Parameters:**
    - `<db_number>`: (Optional) The Redis database number. Defaults to 0.
    - `pool_size`: (Optional) The connection pool size.

### SQL (`postgres`, `mysql`, `mariadb`)

Uses a database table as a message queue.

- **PostgreSQL:**
    - **Scheme:** `postgres://`
    - **Format:** `postgres://<user>:<password>@<host>:<port>/<dbname>?sslmode=disable`
- **MySQL / MariaDB:**
    - **Scheme:** `mysql://` or `mariadb://`
    - **Format:** `mysql://<user>:<password>@<host>:<port>/<dbname>?<options>`

### Kafka (`kafka`)

- **Scheme:** `kafka://`
- **Format:** `kafka://<broker1_host>:<port>,<broker2_host>:<port>/<topic>?consumer_group=<group_name>`
- **Example:** `kafka://localhost:9092/casbin-updates?consumer_group=casbin-group`
- **Parameters:**
    - `consumer_group`: (Required) The Kafka consumer group name.

### AWS SQS-Only (`sqs`)

Uses SQS for both publishing and subscribing (point-to-point).

- **Scheme:** `sqs://`
- **Format:** `sqs://<queue_host>/<account_id>/<queue_name>?region=<region>`
- **Example:** `sqs://sqs.us-east-1.amazonaws.com/123456789012/my-casbin-queue?region=us-east-1`
- **Parameters:**
    - `region`: (Required) The AWS region.

### AWS SNS+SQS (`snssqs`)

Uses SNS for publishing and SQS for subscribing (standard pub/sub).

- **Scheme:** `snssqs://`
- **Format:** `snssqs://<queue_host>/<account_id>/<queue_name>?region=<region>&topic_arn=<your_topic_arn>`
- **Example:**
  `snssqs://sqs.us-east-1.amazonaws.com/123456789012/my-casbin-queue?region=us-east-1&topic_arn=arn:aws:sns:us-east-1:123456789012:my-casbin-topic`
- **Parameters:**
    - `region`: (Required) The AWS region.
    - `topic_arn`: (Required) The full ARN of the SNS topic to publish to.

### Google Cloud Pub/Sub (`gcpv2`)

- **Scheme:** `gcpv2://`
- **Format:** `gcpv2://<project_id>?subscription=<subscription_name>`
- **Example:** `gcpv2://my-gcp-project?subscription=casbin-sub`
- **Parameters:**
    - `subscription`: (Required) The name of the Pub/Sub subscription. The driver will create it if it doesn't exist.

## WatcherEx

This library also supports `WatcherEx` for more granular policy update notifications.

```go
// Create a new WatcherEx instance
w, err := watcher.NewWatcherEx(context.Background(), connectionURL)
if err != nil {
// ...
}

// Now you can use methods like:
// w.UpdateForAddPolicy(...)
// w.UpdateForRemovePolicy(...)
```
