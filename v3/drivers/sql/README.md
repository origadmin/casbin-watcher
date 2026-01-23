# SQL Driver for Casbin Watcher

This directory contains the SQL driver for `casbin-watcher`, providing support for **MySQL** and **PostgreSQL** (
including MariaDB, which uses the MySQL driver).

This driver uses the `watermill-sql` library to provide a Pub/Sub mechanism over a SQL database.

## Supported Databases

- **MySQL** (Driver name: `mysql`)
- **PostgreSQL** (Driver name: `postgres`)
- **MariaDB** (Driver name: `mariadb`, uses the `mysql` driver)

**Note:** For SQLite support, please use the standalone `sqlite` driver in the parent directory.

## Configuration

The driver is configured using a database connection URL. The scheme of the URL determines which database driver is
used.

### MySQL / MariaDB

The driver will automatically reconstruct the DSN from the URL.

**URL Format:**

```
mysql://user:password@tcp(host:port)/dbname?query_params
```

or

```
mariadb://user:password@tcp(host:port)/dbname?query_params
```

**Example:**

```
mysql://root:password@tcp(127.0.0.1:3306)/casbin
```

### PostgreSQL

For PostgreSQL, the URL is passed directly as the DSN.

**URL Format:**

```
postgres://user:password@host:port/dbname?query_params
```

**Example:**

```
postgres://user:password@localhost:5432/casbin?sslmode=disable
```

### Table Creation

The driver uses `watermill`'s schema adapters to automatically create the necessary tables (e.g., `watermill_messages`,
`watermill_offsets`) for storing messages and offsets if they do not already exist.

## Usage Example

```go
import (
    "context"
    "log"

    "github.com/casbin/casbin/v2"
    "github.com/origadmin/casbin-watcher/v3"
    _ "github.com/origadmin/casbin-watcher/v3/drivers/sql" // Register the driver
)

func main() {
    // Example for PostgreSQL
    // The topic for policy updates is "casbin_updates".
    connectionURL := "postgres://user:pass@localhost:5432/casbin_db?sslmode=disable"
    
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
    
    // Policy changes will now be broadcast via the SQL database.
}
```
