# Google Cloud Pub/Sub Driver for Casbin Watcher

This directory contains the Google Cloud Pub/Sub driver for `casbin-watcher`. It is based on the `watermill-googlecloud`
library.

## How it Works

This driver uses Google Cloud Pub/Sub for broadcasting policy updates.

- **Publisher**: Sends messages to a specified Google Cloud Pub/Sub topic.
- **Subscriber**: Receives messages from a subscription attached to that topic.

The driver will automatically create the topic and the subscription if they do not exist.

## Authentication

The driver uses the default Google Cloud SDK credential chain. This means you can configure credentials via:

1. Setting the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of your service account JSON key file.
2. Running on a GCP service (like GCE, GKE, Cloud Run) that has a service account attached.
3. Running `gcloud auth application-default login` on your local machine.

## Configuration

The driver is configured using a URL.

### URL Format

```
gcpv2://<project_id>?subscription_name=<sub_name>&<other_parameters>
```

- `<project_id>`: **(Required)** Your Google Cloud Project ID.
- The topic for publishing and subscribing is passed from the `watcher.NewWatcher` call, not from the URL.

### Configuration Parameters

| Parameter                  | Type      | Default   | Description                                                                                                                                                                                                       | Example                        |
|----------------------------|-----------|-----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| `subscription_name`        | `string`  | `(topic)` | The name of the Pub/Sub subscription to use. If not provided, it defaults to the topic name. **It is highly recommended to provide a unique name** for each service to ensure it receives all messages (fan-out). | `subscription_name=my-app-sub` |
| `max_outstanding_messages` | `integer` | `100`     | Maximum number of messages that the subscriber will hold in memory at a time. Adjust this value based on your message size and available memory.                                                                  | `max_outstanding_messages=200` |
| `num_goroutines`           | `integer` | `10`      | Number of goroutines used to process messages. Increase this value to improve throughput.                                                                                                                         | `num_goroutines=20`            |

### Subscription Naming

- **Default (Not Recommended for Fan-out)**: If `subscription_name` is omitted, all watchers subscribing to the same
  topic (e.g., `casbin_rules`) will try to use the same subscription name (`casbin_rules`). This creates a **competing
  consumer** pattern where only one watcher instance will receive a given message.
- **Recommended**: Provide a unique `subscription_name` for each distinct service that uses the watcher. This creates a
  **fan-out** pattern where each service gets its own subscription and receives a copy of all messages published to the
  topic.

## Usage Example

```go
import (
"github.com/casbin/casbin/v2"
"github.com/origadmin/casbin-watcher/v3"
_ "github.com/origadmin/casbin-watcher/v3/drivers/gcpv2" // Register the driver
)

func main() {
// Recommended setup: specify a unique subscription name for your service.
// The topic "casbin-policy-updates" is passed as the second argument.
w, err := watcher.NewWatcher(
"gcpv2://my-gcp-project-id?subscription_name=my-app-casbin-sub&max_outstanding_messages=200&num_goroutines=20",
"casbin-policy-updates",
)
if err != nil {
panic(err)
}

e, err := casbin.NewEnforcer("model.conf", "policy.csv")
if err != nil {
panic(err)
}

err = e.SetWatcher(w)
if err != nil {
panic(err)
}

// ...
}
```
