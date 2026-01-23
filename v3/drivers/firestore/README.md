# Google Cloud Firestore Driver for Casbin Watcher

This directory contains the Google Cloud Firestore (`firestore`) driver for `casbin-watcher`. This driver uses
Firestore, a flexible, scalable NoSQL document database, to broadcast policy updates.

## How it Works

The `firestore` driver is built on Watermill's `firestore` Pub/Sub implementation.

- **Storage**: It uses a Firestore collection to store messages. Each message is a document in the collection.
- **Authentication**: The driver uses Application Default Credentials (ADC) to authenticate with Google Cloud. Ensure
  your application is running in an environment where ADC is configured (e.g., a GCP VM, GKE, or by using
  `gcloud auth application-default login`).
- **Subscription Names**: You can provide a `subscription_id` in the URL. If not provided, a unique subscription name
  will be generated based on the topic and a UUID.
- **Use Case**: This driver is suitable for applications already running on Google Cloud Platform that prefer using
  Firestore for messaging.

## Configuration

The driver is configured using a URL.

### URL Format

```
firestore://?project_id=your-gcp-project-id&subscription_id=your-subscription-id
```

- **Scheme**: The scheme must be `firestore`.
- **Parameters**: Configuration options are provided via query parameters.

### Configuration Parameters

| Parameter         | Type     | Default        | Description                                                                      | Example                         |
|-------------------|----------|----------------|----------------------------------------------------------------------------------|---------------------------------|
| `project_id`      | `string` | (none)         | **Required.** Your Google Cloud project ID.                                      | `project_id=my-awesome-project` |
| `subscription_id` | `string` | `topic-<UUID>` | The ID of the Firestore subscription. If omitted, a unique ID will be generated. | `subscription_id=casbin-sub`    |

### Usage Example

```go
import (
"context"
"log"

"github.com/casbin/casbin/v2"
"github.com/origadmin/casbin-watcher/v3"
_ "github.com/origadmin/casbin-watcher/v3/drivers/firestore" // Register the driver
)

func main() {
// The topic for policy updates is "casbin_updates".
connectionURL := "firestore://?project_id=my-gcp-project&subscription_id=casbin-watcher-sub"

w, err := watcher.NewWatcher(context.Background(), connectionURL, "casbin_updates")
if err != nil {
log.Fatalf("Failed to create watcher: %v", err)
}

e, err := casbin.NewEnforcer("model.conf", "policy.csv")
if err != nil {
log.Fatalf("Failed to create enforcer: %v", err)
}

err = e.SetWatcher(w); err != nil {
log.Fatalf("Failed to set watcher: %v", err)
}

// Policy changes will now be broadcast via Firestore.
}
```
