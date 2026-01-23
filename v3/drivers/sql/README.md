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

### PostgreSQL

```
postgres://user:pass@localhost:5432/casbin_db?sslmode=disable
```

### MySQL

```
mysql://root:password@tcp(127.0.0.1:3306)/casbin
```

The topic for policy updates is taken from the URL path or defaults to `casbin-policy-updates`. You can also specify it
using the `WithTopic` option.
