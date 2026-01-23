package redisstream

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("redisstream", &Driver{})
}

// Driver implements the watcher.Driver interface for Redis Streams.
type Driver struct{}

// NewPubSub creates a new PubSub for Redis Streams.
func (d *Driver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	opts, consumerGroup, err := parseRedisURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close() // Ensure client is closed on ping failure.
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:     rdb,
			Marshaller: redisstream.DefaultMarshallerUnmarshaller{},
		},
		logger,
	)
	if err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to create redis publisher: %w", err)
	}

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        rdb,
			Unmarshaller:  redisstream.DefaultMarshallerUnmarshaller{},
			ConsumerGroup: consumerGroup,
		},
		logger,
	)
	if err != nil {
		_ = publisher.Close()
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to create redis subscriber: %w", err)
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
		rdb:        rdb,
	}, nil
}

type pubSub struct {
	publisher  *redisstream.Publisher
	subscriber *redisstream.Subscriber
	rdb        *redis.Client
}

func (r *pubSub) Publish(topic string, messages ...*message.Message) error {
	return r.publisher.Publish(topic, messages...)
}

func (r *pubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return r.subscriber.Subscribe(ctx, topic)
}

func (r *pubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, r.publisher.Close())
	allErrors = multierr.Append(allErrors, r.subscriber.Close())
	allErrors = multierr.Append(allErrors, r.rdb.Close())
	return allErrors
}

func parseRedisURL(u *url.URL) (*redis.Options, string, error) {
	if u.Host == "" {
		return nil, "", fmt.Errorf("redis host is not specified in URL")
	}

	opts := &redis.Options{
		Addr: u.Host,
	}

	if u.User != nil {
		if password, ok := u.User.Password(); ok {
			opts.Password = password
		}
	}

	// The path is used for the DB number, not the topic.
	if u.Path != "" && u.Path != "/" {
		dbStr := u.Path[1:]
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, "", fmt.Errorf("invalid redis db in URL path: %s", dbStr)
		}
		opts.DB = db
	}

	query := u.Query()
	if poolSizeStr := query.Get("pool_size"); poolSizeStr != "" {
		poolSize, err := strconv.Atoi(poolSizeStr)
		if err != nil {
			return nil, "", fmt.Errorf("invalid pool_size query parameter: %w", err)
		}
		opts.PoolSize = poolSize
	}

	consumerGroup := query.Get("consumer_group")
	if consumerGroup == "" {
		// Generate a unique consumer group name if not provided.
		// This ensures that each watcher instance acts as an independent consumer.
		consumerGroup = "casbin-watcher-" + watermill.NewUUID()
	}

	return opts, consumerGroup, nil
}
