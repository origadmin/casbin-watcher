package sql

import (
	"context"
	stdSQL "database/sql"
	"fmt"
	"net/url"

	// Import SQL drivers for side effects
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("postgres", &Driver{})
	watcher.RegisterDriver("mysql", &Driver{})
	watcher.RegisterDriver("mariadb", &Driver{})

}

// Driver implements the watcher.Driver interface for SQL databases.
type Driver struct{}

// NewPubSub creates a new PubSub for SQL.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	// The URL scheme is the SQL driver name (e.g., "mysql", "postgres").
	if u.Scheme == "mariadb" {
		u.Scheme = "mysql"
	}
	db, err := stdSQL.Open(u.Scheme, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open sql database: %w", err)
	}

	// Ping the database to ensure the connection is valid.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to sql database: %w", err)
	}

	beginner := sql.BeginnerFromStdSQL(db)
	schemaAdapter := d.getSchemaAdapter(u.Scheme)
	offsetsAdapter := d.getOffsetsAdapter(u.Scheme)

	publisher, err := sql.NewPublisher(
		beginner,
		sql.PublisherConfig{
			SchemaAdapter: schemaAdapter,
		},
		logger,
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create sql publisher: %w", err)
	}

	subscriber, err := sql.NewSubscriber(
		beginner,
		sql.SubscriberConfig{
			SchemaAdapter:    schemaAdapter,
			OffsetsAdapter:   offsetsAdapter,
			InitializeSchema: true, // Automatically create tables if they don't exist
		},
		logger,
	)
	if err != nil {
		_ = publisher.Close()
		_ = db.Close()
		return nil, fmt.Errorf("failed to create sql subscriber: %w", err)
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
		db:         db,
	}, nil
}

func (d *Driver) getSchemaAdapter(scheme string) sql.SchemaAdapter {
	switch scheme {
	case "postgres":
		return sql.DefaultPostgreSQLSchema{}
	case "mysql":
		return sql.DefaultMySQLSchema{}
	default:
		// This should not happen as we only register for "postgres" and "mysql".
		// Fallback to PostgreSQL schema.
		return sql.DefaultPostgreSQLSchema{}
	}
}

func (d *Driver) getOffsetsAdapter(scheme string) sql.OffsetsAdapter {
	switch scheme {
	case "postgres":
		return sql.DefaultPostgreSQLOffsetsAdapter{}
	case "mysql":
		return sql.DefaultMySQLOffsetsAdapter{}
	default:
		// Fallback to PostgreSQL offsets adapter.
		return sql.DefaultPostgreSQLOffsetsAdapter{}
	}
}

type pubSub struct {
	publisher  *sql.Publisher
	subscriber *sql.Subscriber
	db         *stdSQL.DB
}

func (p *pubSub) Publish(topic string, messages ...*message.Message) error {
	return p.publisher.Publish(topic, messages...)
}

func (p *pubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return p.subscriber.Subscribe(ctx, topic)
}

func (p *pubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, p.publisher.Close())
	allErrors = multierr.Append(allErrors, p.subscriber.Close())
	allErrors = multierr.Append(allErrors, p.db.Close())
	return allErrors
}
