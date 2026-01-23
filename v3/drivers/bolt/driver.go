package bolt

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-bolt/pkg/bolt"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.etcd.io/bbolt"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("bolt", &Driver{})
}

// Driver implements the watcher.Driver interface for BoltDB.
type Driver struct{}

// NewPubSub creates a new PubSub for BoltDB.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, _ watermill.LoggerAdapter) (watcher.PubSub, error) {
	dbFilePath := u.Query().Get("path")
	if dbFilePath == "" {
		return nil, fmt.Errorf("bolt driver requires a 'path' query parameter specifying the database file")
	}

	db, err := bbolt.Open(dbFilePath, os.ModePerm, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %w", err)
	}

	// The bucket name is derived from the URL path.
	// If path is empty or "/", use a default bucket name.
	bucketName := "casbin_watcher_events"
	if u.Path != "" && u.Path != "/" {
		bucketName = strings.TrimPrefix(u.Path, "/")
	}

	commonConfig := bolt.CommonConfig{
		Bucket: []bolt.BucketName{bolt.BucketName(bucketName)},
	}

	publisher, err := bolt.NewPublisher(db, bolt.PublisherConfig{
		Common: commonConfig,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create bolt publisher: %w", err)
	}

	subscriber, err := bolt.NewSubscriber(db, bolt.SubscriberConfig{
		Common: commonConfig,
	})
	if err != nil {
		_ = publisher.Close()
		_ = db.Close()
		return nil, fmt.Errorf("failed to create bolt subscriber: %w", err)
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
		db:         db,
	}, nil
}

type pubSub struct {
	publisher  bolt.Publisher
	subscriber *bolt.Subscriber
	db         *bbolt.DB
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
