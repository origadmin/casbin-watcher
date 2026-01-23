# IO Driver for Casbin Watcher

This directory contains the IO (`io`) driver for `casbin-watcher`. This driver uses standard `io.ReadCloser` and
`io.WriteCloser` interfaces for publishing and subscribing messages.

## How it Works

The `io` driver is built on Watermill's `io` Pub/Sub implementation.

- **Default Behavior**: By default, the driver uses `os.Stdout` for publishing and `os.Stdin` for subscribing.
- **File-Based**: You can specify a file path using the `path` query parameter. In this mode, the driver will read from
  and write to the same file. This is useful for simple, file-based inter-process communication on the same machine.
- **Simplicity**: This driver is primarily useful for debugging, testing, or basic IPC.
- **No Topics**: The `io` driver does not inherently support topics. All messages published will go to the configured
  writer, and all messages subscribed will come from the configured reader.

## Configuration

The driver is configured using a URL.

### URL Format

```
io://?path=/path/to/your/file
```

- **Scheme**: The scheme must be `io`.
- **Parameters**: The file path is configured via a query parameter.

### Configuration Parameters

| Parameter | Type     | Default        | Description                                                                                                                     | Example             |
|-----------|----------|----------------|---------------------------------------------------------------------------------------------------------------------------------|---------------------|
| `path`    | `string` | (stdin/stdout) | If provided, the driver will use this file for both reading and writing. If omitted, it defaults to `os.Stdin` and `os.Stdout`. | `path=./events.log` |

### Usage Example

```go
import (
    "context"
    "log"

    "github.com/casbin/casbin/v2"
    "github.com/origadmin/casbin-watcher/v3"
    _ "github.com/origadmin/casbin-watcher/v3/drivers/io" // Register the driver
)

func main() {
    // Example 1: Use standard input/output
    // connectionURL_stdio := "io://"
    // w, _ := watcher.NewWatcher(context.Background(), connectionURL_stdio, "casbin_updates")

    // Example 2: Use a file
    connectionURL_file := "io://?path=./casbin_events.log"
    w, err := watcher.NewWatcher(context.Background(), connectionURL_file, "casbin_updates")
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
    
    // Policy changes will now be written to the specified file or stdout.
}
```
