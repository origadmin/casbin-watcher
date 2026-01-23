# SQLite Driver for Casbin Watcher

This directory contains the standalone SQLite driver for `casbin-watcher`.

This driver uses the `watermill-sql` library to provide a Pub/Sub mechanism over a SQLite database. It relies on the
standard `github.com/mattn/go-sqlite3` driver.

## Configuration

The driver is configured using a URL. The format requires the path to the database file.

### URL Format

```
sqlite3:///path/to/your_database.db
```

- The scheme must be `sqlite3`.
- The path should be an absolute path to the database file.

**Note on Windows Paths:** For Windows paths, the format should still use forward slashes and include a leading slash
before the drive letter. The driver will correctly parse it.

```
sqlite3:///D:/path/to/your_database.db
```

### Table Creation

The driver uses `watermill`'s `DefaultSQLiteSchema` to automatically create the necessary table (e.g.,
`watermill_casbin-updates`) for storing messages if it does not already exist.

## Usage Example

```go
import (
"github.com/casbin/casbin/v2"
"github.com/origadmin/casbin-watcher/v3"
_ "github.com/origadmin/casbin-watcher/v3/drivers/sqlite" // Register the driver
)

func main() {
// The URL points to the SQLite database file.
// The topic "casbin_rules" will be used as the table name suffix.
w, err := watcher.NewWatcher("sqlite3:///D:/temp/test.db?topic=casbin_rules")
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
