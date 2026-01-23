package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("mysql", &Driver{dbType: "mysql"})
	watcher.RegisterDriver("postgres", &Driver{dbType: "postgres"})
	watcher.RegisterDriver("mariadb", &Driver{dbType: "mysql"}) // MariaDB uses the MySQL driver
}

// Driver handles SQL-based Pub/Sub for MySQL and PostgreSQL.
type Driver struct {
	dbType string
}

// NewPubSub creates a new Pub/Sub for the configured SQL database.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	dsn, err := parseSQLURL(u, d.dbType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sql url: %w", err)
	}

	db, err := sql.Open(d.dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sql database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to sql database: %w", err)
	}

	var schemaAdapter watermillsql.SchemaAdapter
	switch d.dbType {
	case "mysql":
		schemaAdapter = watermillsql.DefaultMySQLSchema{
			GenerateMessagesTableName: func(topic string) string {
				// Use backticks for MySQL to avoid issues with reserved keywords
				return fmt.Sprintf("`watermill_%s`", topic)
			},
		}
	case "postgres":
		schemaAdapter = watermillsql.DefaultPostgreSQLSchema{}
	default:
		// This case should not be reached due to the init function.
		_ = db.Close()
		return nil, fmt.Errorf("unsupported db type: %s", d.dbType)
	}

	publisher, err := watermillsql.NewPublisher(
		db,
		watermillsql.PublisherConfig{
			SchemaAdapter: schemaAdapter,
		},
		logger,
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create sql publisher: %w", err)
	}

	subscriber, err := watermillsql.NewSubscriber(
		db,
		watermillsql.SubscriberConfig{
			SchemaAdapter:  schemaAdapter,
			ConsumerGroup:  "casbin-watcher",
			PollInterval:   time.Second,
			ResendInterval: time.Second * 5,
		},
		logger,
	)
	if err != nil {
		_ = publisher.Close()
		_ = db.Close()
		return nil, fmt.Errorf("failed to create sql subscriber: %w", err)
	}

	return &sqlPubSub{
		publisher:  publisher,
		subscriber: subscriber,
		db:         db,
	}, nil
}

type sqlPubSub struct {
	publisher  *watermillsql.Publisher
	subscriber *watermillsql.Subscriber
	db         *sql.DB
}

func (s *sqlPubSub) Publish(topic string, messages ...*message.Message) error {
	return s.publisher.Publish(topic, messages...)
}

func (s *sqlPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, topic)
}

func (s *sqlPubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, s.publisher.Close())
	allErrors = multierr.Append(allErrors, s.subscriber.Close())
	allErrors = multierr.Append(allErrors, s.db.Close())
	return allErrors
}

func parseSQLURL(u *url.URL, dbType string) (string, error) {
	switch dbType {
	case "postgres":
		// For postgres, the URL format is usually directly compatible.
		return u.String(), nil
	case "mysql":
		// For mysql, the DSN format is user:password@tcp(host)/dbname?query
		// We need to reconstruct it from the URL.
		password, _ := u.User.Password()
		dsn := fmt.Sprintf("%s:%s@tcp(%s)%s", u.User.Username(), password, u.Host, u.Path)
		if u.RawQuery != "" {
			dsn += "?" + u.RawQuery
		}
		return dsn, nil
	default:
		return "", fmt.Errorf("unsupported db type for DSN parsing: %s", dbType)
	}
}
