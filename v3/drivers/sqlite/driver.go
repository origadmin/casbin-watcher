package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/mattn/go-sqlite3" // Standard SQLite driver
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("sqlite3", &Driver{})
}

// Driver for SQLite.
type Driver struct{}

// NewPubSub creates a new Pub/Sub for SQLite.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	dsn := parseSQLiteURL(u)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to sqlite database: %w", err)
	}

	// Use the correct SchemaAdapter for SQLite.
	schemaAdapter := watermillsql.DefaultSQLiteSchema{}

	publisher, err := watermillsql.NewPublisher(
		db,
		watermillsql.PublisherConfig{
			SchemaAdapter: schemaAdapter,
		},
		logger,
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create sqlite publisher: %w", err)
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
		return nil, fmt.Errorf("failed to create sqlite subscriber: %w", err)
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
		db:         db,
	}, nil
}

type pubSub struct {
	publisher  *watermillsql.Publisher
	subscriber *watermillsql.Subscriber
	db         *sql.DB
}

func (s *pubSub) Publish(topic string, messages ...*message.Message) error {
	return s.publisher.Publish(topic, messages...)
}

func (s *pubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, topic)
}

func (s *pubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, s.publisher.Close())
	allErrors = multierr.Append(allErrors, s.subscriber.Close())
	allErrors = multierr.Append(allErrors, s.db.Close())
	return allErrors
}

// parseSQLiteURL extracts the file path from a URL.
// For SQLite, the DSN is the path to the database file.
// A URL like `sqlite3:///path/to/your.db` will be parsed to `/path/to/your.db`.
// On Windows, `sqlite3:///D:/path/to/your.db` becomes `D:/path/to/your.db`.
func parseSQLiteURL(u *url.URL) string {
	// For Windows paths, the leading `/` needs to be removed.
	// For example, /D:/path/to/db.sqlite -> D:/path/to/db.sqlite
	if len(u.Path) > 2 && u.Path[0] == '/' && u.Path[2] == ':' {
		return strings.TrimPrefix(u.Path, "/")
	}
	return u.Path
}
