# SQLite Driver for Casbin Watcher

This directory contains the SQLite driver for `casbin-watcher`.

This driver uses the `wmsqlitemodernc` package from `watermill-sqlite`, which is a pure Go implementation and does not
require CGO.

## Configuration

The driver is configured using a URL.

### URL Format

```
sqlite://?path=/path/to/your_database.db
```

- **Scheme**: The scheme must be `sqlite`.
- **Parameters**: The database file path is configured via a query parameter.

### Configuration Parameters

| Parameter | Type     | Default | Description                                         | Example            |
|-----------|----------|---------|-----------------------------------------------------|--------------------|
| `path`    | `string` | (none)  | **Required.** The path to the SQLite database file. | `path=./my_app.db` |

### Table Creation

The driver will automatically create the necessary tables for storing messages and offsets if they do not already exist.

## Usage Example

```go
import (
"context"
"log"

"github.com/casbin/casbin/v2"
"github.com/origadmin/casbin-watcher/v3"
_ "github.com/origadmin/casbin-watcher/v3/drivers/sqlite" // Register the driver
)

func main() {
// The topic for policy updates is "casbin_updates".
connectionURL := "sqlite://?path=./casbin_events.db"

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

// Policy changes will now be broadcast via the SQLite database.
}
```
