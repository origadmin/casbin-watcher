package sqlite

import (
	"context"
	stdSQL "database/sql"
	"fmt"
	"net/url"
	"strings"

	"go.uber.org/multierr"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sqlite/wmsqlitemodernc"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/origadmin/casbin-watcher/v3"

	_ "modernc.org/sqlite" // modernc.org/sqlite driver for wmsqlitemodernc
)

func init() {
	watcher.RegisterDriver("sqlite", &Driver{})
}

// Driver for SQLite.
type Driver struct{}

// NewPubSub creates a new Pub/Sub for SQLite.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	// Determine the database file path.
	// If the URL path is provided, use it. Otherwise, check the 'path' query parameter.
	// If neither, default to an in-memory database.
	dbFilePath := u.Path
	if dbFilePath == "" || dbFilePath == "/" {
		dbFilePath = u.Query().Get("path")
		if dbFilePath == "" {
			dbFilePath = ":memory:" // Default to in-memory if no path is specified
		}
	} else {
		// Remove leading slash if present, for consistency with file paths.
		dbFilePath = strings.TrimPrefix(dbFilePath, "/")
	}

	// Build the DSN string.
	// All query parameters from the URL are passed as SQLite pragmas.
	dsn := fmt.Sprintf("file:%s", dbFilePath)
	if len(u.Query()) > 0 {
		dsn += "?" + u.Query().Encode()
	}

	db, err := stdSQL.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open modernc sqlite database: %w", err)
	}
	// modernc.org/sqlite limitation
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to modernc sqlite database: %w", err)
	}

	publisher, err := wmsqlitemodernc.NewPublisher(
		db,
		wmsqlitemodernc.PublisherOptions{
			InitializeSchema: true,
			Logger:           logger,
		},
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create modernc sqlite publisher: %w", err)
	}

	subscriber, err := wmsqlitemodernc.NewSubscriber(
		db,
		wmsqlitemodernc.SubscriberOptions{
			InitializeSchema: true,
			Logger:           logger,
		},
	)
	if err != nil {
		_ = publisher.Close()
		_ = db.Close()
		return nil, fmt.Errorf("failed to create modernc sqlite subscriber: %w", err)
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
		db:         db,
	}, nil
}

type pubSub struct {
	publisher  message.Publisher
	subscriber message.Subscriber
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
