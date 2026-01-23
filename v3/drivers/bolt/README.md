# BoltDB Driver for Casbin Watcher

This directory contains the BoltDB (`bolt`) driver for `casbin-watcher`. This driver uses BoltDB, an embedded key-value
database, to facilitate communication between watchers within the same process or on the same machine.

## How it Works

The `bolt` driver is built on Watermill's `bolt` Pub/Sub implementation.

- **Storage**: It uses a single file on disk as its database, specified by the `path` query parameter.
- **Bucket**: Messages are stored in a BoltDB bucket. The name of this bucket is derived from the URL's path component.
- **Communication**: Publishers write messages to a topic (which corresponds to a bucket in BoltDB), and subscribers
  read from it.
- **Use Case**: This driver is ideal for scenarios where you need a simple, file-based, and persistent communication
  mechanism without setting up a separate broker service. It's a step up from the in-memory driver, providing durability
  across application restarts.

**Note**: Since BoltDB locks the database file, only one process can access the database at a time. This driver is not
suitable for broadcasting updates across multiple, independent processes running simultaneously.

## Configuration

The driver is configured using a URL.

### URL Format

```
bolt:///your_bucket_name?path=/path/to/your/database.db
```

- **Scheme**: The scheme must be `bolt`.
- **Bucket Name**: The name of the BoltDB bucket is specified in the **path** part of the URL (e.g.,
  `/your_bucket_name`). If the path is empty or `/`, a default bucket name (`casbin_watcher_events`) will be used.
- **Parameters**: The database file path is configured via a query parameter.

### Configuration Parameters

| Parameter | Type     | Default | Description                                                     | Example                            |
|-----------|----------|---------|-----------------------------------------------------------------|------------------------------------|
| `path`    | `string` | (none)  | **Required.** The absolute or relative path to the BoltDB file. | `path=/var/data/casbin_watcher.db` |

## Usage Example

### Basic Connection

```
bolt:///casbin_updates?path=./casbin_events.db
```

### With Default Bucket

```
bolt://?path=./casbin_events.db
```
