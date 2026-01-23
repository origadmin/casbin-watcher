# Casbin Watermill Watcher

[![Go Report Card](https://goreportcard.com/badge/github.com/casbin/watermill-adapter)](https://goreportcard.com/badge/github.com/casbin/watermill-adapter)
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
_ "github.com/origadmin/casbin-watcher/v3/drivers/nats" // Corrected import path
)

func main() {
// The connection URL for the desired driver.
// See the driver's README for configuration details.
connectionURL := "nats://localhost:4222/casbin_updates" // Corrected NATS URL

// Create a new watcher.
w, err := watcher.NewWatcher(context.Background(), connectionURL) // Removed topic argument
if err != nil {
log.Fatalf("Failed to create watcher: %v", err)
}

// Initialize the enforcer.
e, err := casbin.NewEnforcer("model.conf", "policy.csv")
if err != nil {
log.Fatalf("Failed to create enforcer: %v", err)
}

// Set the watcher for the enforcer.
err = e.SetWatcher(w)
if err != nil {
log.Fatalf("Failed to set watcher: %v", err)
}

// The watcher will now automatically update other enforcer instances
// when the policy changes.
// For example, after e.SavePolicy() or e.AddPolicy(), etc.
}
```

## Supported Drivers

This section lists all Watermill Pub/Sub backends, indicating their implementation status within this `casbin-watcher`
repository.

| Driver Name                                 | Scheme(s)                 | Underlying Watermill Package                                     | Status          |
|---------------------------------------------|---------------------------|------------------------------------------------------------------|-----------------|
| [**AWS (SQS/SNS)**](./drivers/aws)          | `sqs://`, `snssqs://`     | `github.com/ThreeDotsLabs/watermill-aws`                         | Implemented     |
| [**BoltDB**](./drivers/bolt)                | `bolt://`                 | `github.com/ThreeDotsLabs/watermill-bolt`                        | Implemented     |
| [**etcd**](./drivers/etcd)                  | `etcd://`                 | Custom implementation (using `go.etcd.io/etcd/client/v3`)        | Implemented     |
| [**Firestore**](./drivers/firestore)        | `firestore://`            | `github.com/ThreeDotsLabs/watermill-firestore`                   | Implemented     |
| [**Google Cloud Pub/Sub**](./drivers/gcpv2) | `gcpv2://`                | `github.com/ThreeDotsLabs/watermill-googlecloud/v2`              | Implemented     |
| [**HTTP**](./drivers/http)                  | `http://`                 | `github.com/ThreeDotsLabs/watermill-http/v2`                     | Implemented     |
| [**IO (Stdin/Stdout/File)**](./drivers/io)  | `io://`                   | `github.com/ThreeDotsLabs/watermill-io`                          | Implemented     |
| [**Kafka**](./drivers/kafka)                | `kafka://`                | `github.com/ThreeDotsLabs/watermill-kafka/v3`                    | Implemented     |
| [**NATS**](./drivers/nats)                  | `nats://`                 | `github.com/ThreeDotsLabs/watermill-nats/v2`                     | Implemented     |
| [**RabbitMQ**](./drivers/rabbitmq)          | `rabbitmq://`             | `github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp`            | Implemented     |
| [**Redis Streams**](./drivers/redisstream)  | `redisstream://`          | `github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream` | Implemented     |
| [**SQL (PostgreSQL/MySQL)**](./drivers/sql) | `postgres://`, `mysql://` | `github.com/ThreeDotsLabs/watermill-sql/v4`                      | Implemented     |
| [**SQLite (modernc)**](./drivers/sqlite)    | `sqlite://`               | `github.com/ThreeDotsLabs/watermill-sqlite/wmsqlitemodernc`      | Implemented     |
| **AMQP 1.0**                                | `amqp10://`               | `github.com/kahowell/watermill-amqp10`                           | Not Implemented |
| **Apache Pulsar**                           | `pulsar://`               | `github.com/AlexCuse/watermill-pulsar`                           | Not Implemented |
| **Apache RocketMQ**                         | `rocketmq://`             | `github.com/yflau/watermill-rocketmq`                            | Not Implemented |
| **CockroachDB**                             | `cockroachdb://`          | `github.com/cockroachdb/watermill-crdb`                          | Not Implemented |
| **Ensign**                                  | `ensign://`               | `github.com/rotationalio/watermill-ensign`                       | Not Implemented |
| **Google Cloud (HTTP Push)**                | `gcp-http-push://`        | `github.com/dentech-floss/watermill-googlecloud-http`            | Not Implemented |
| **MongoDB**                                 | `mongodb://`              | `github.com/cunyat/watermill-mongodb`                            | Not Implemented |
| **MQTT**                                    | `mqtt://`                 | `github.com/perfect13/watermill-mqtt`                            | Not Implemented |
| **NSQ**                                     | `nsq://`                  | `github.com/chennqqi/watermill-nsq`                              | Not Implemented |
| **Redis (ZSET)**                            | `rediszset://`            | `github.com/stong1994/watermill-rediszset`                       | Not Implemented |
| **SQLite (zombiezen)**                      | `sqlite-zombiezen://`     | `github.com/ThreeDotsLabs/watermill-sqlite/wmsqlitezombiezen`    | Not Implemented |

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
