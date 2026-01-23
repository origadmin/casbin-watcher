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

The driver uses `watermill`'s schema adapters to automatically create the necessary table (e.g.,
`watermill_casbin-updates`) for storing messages if it does not already exist.

## Usage Example

```go
import (
"github.com/casbin/casbin/v2"
"github.com/origadmin/casbin-watcher/v3"
_ "github.com/origadmin/casbin-watcher/v3/drivers/sql" // Register the driver
)

func main() {
// Example for PostgreSQL
w, err := watcher.NewWatcher("postgres://user:pass@localhost:5432/casbin_db?sslmode=disable&topic=casbin_rules")
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
