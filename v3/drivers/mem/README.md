# In-Memory Driver for Casbin Watcher

This directory contains the in-memory (`mem`) driver for `casbin-watcher`. This driver is ideal for testing,
development, or single-instance applications where policy updates do not need to be broadcast across a network.

## How it Works

The `mem` driver uses Watermill's `gochannel` Pub/Sub, which operates entirely within the application's memory space. It
can be configured in two modes:

1. **Shared (Default)**: All watcher instances created with `shared=true` (or with the parameter omitted) will use a
   single, global Pub/Sub instance. This allows different components within the same application to communicate policy
   updates.
2. **Isolated**: If `shared=false` is specified, each watcher instance will receive its own private Pub/Sub instance.
   This is useful for creating isolated test environments.

## Configuration

The driver is configured using a URL.

### URL Format

```
mem:///topic?shared=true&buffer_size=1024
```

- **Topic**: The topic for policy updates, provided in the `path` part of the URL.
- **Parameters**: Additional options are configured via query parameters.

### Configuration Parameters

| Parameter     | Type      | Default | Description                                                                                          | Example            |
|---------------|-----------|---------|------------------------------------------------------------------------------------------------------|--------------------|
| `shared`      | `boolean` | `true`  | If `true`, uses a single global instance for all watchers. If `false`, creates an isolated instance. | `shared=false`     |
| `buffer_size` | `integer` | `0`     | The size of the buffer for the underlying Go channel. A size of `0` means the channel is unbuffered. | `buffer_size=1024` |

## Usage Example

### Shared Instance

```
mem:///casbin_updates
```

### Isolated Instance

```
mem:///casbin_updates?shared=false
```

### With Buffer Size

```
mem:///casbin_updates?buffer_size=1024
```
