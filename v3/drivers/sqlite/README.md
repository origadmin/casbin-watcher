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

```
sqlite://?path=./casbin_events.db
```

The topic for policy updates is taken from the URL path or defaults to `casbin-policy-updates`. You can also specify it
using the `WithTopic` option.
