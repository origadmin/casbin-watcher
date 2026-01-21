package redis

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
	watcher.RegisterDriver("redis", &Driver{})
}

type Driver struct{}

func (d *Driver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	opts, err := parseRedisURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client: rdb,
		},
		logger,
	)
	if err != nil {
		// If publisher creation fails, ensure rdb is closed.
		if closeErr := rdb.Close(); closeErr != nil {
			logger.Error("failed to close Redis client after publisher creation failed", closeErr, nil)
		}
		return nil, fmt.Errorf("failed to create redis publisher: %w", err)
	}

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client: rdb,
		},
		logger,
	)
	if err != nil {
		// If subscriber creation fails, ensure publisher and rdb are closed.
		if closeErr := publisher.Close(); closeErr != nil {
			logger.Error("failed to close Redis publisher after subscriber creation failed", closeErr, nil)
		}
		if closeErr := rdb.Close(); closeErr != nil {
			logger.Error("failed to close Redis client after subscriber creation failed", closeErr, nil)
		}
		return nil, fmt.Errorf("failed to create redis subscriber: %w", err)
	}

	return &redisPubSub{
		publisher:  publisher,
		subscriber: subscriber,
		rdb:        rdb,
	}, nil
}

type redisPubSub struct {
	publisher  *redisstream.Publisher
	subscriber *redisstream.Subscriber
	rdb        *redis.Client
}

func (r *redisPubSub) Publish(topic string, messages ...*message.Message) error {
	return r.publisher.Publish(topic, messages...)
}

func (r *redisPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return r.subscriber.Subscribe(ctx, topic)
}

func (r *redisPubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, r.publisher.Close())
	allErrors = multierr.Append(allErrors, r.subscriber.Close())
	allErrors = multierr.Append(allErrors, r.rdb.Close())
	return allErrors
}

func parseRedisURL(u *url.URL) (*redis.Options, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("redis host is not specified in URL")
	}

	opts := &redis.Options{
		Addr: u.Host,
	}

	if u.User != nil {
		if password, ok := u.User.Password(); ok {
			opts.Password = password
		}
	}

	if u.Path != "" && u.Path != "/" {
		dbStr := u.Path[1:]
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid redis db in URL path: %s", dbStr)
		}
		opts.DB = db
	}

	query := u.Query()
	if poolSizeStr := query.Get("pool_size"); poolSizeStr != "" {
		poolSize, err := strconv.Atoi(poolSizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid pool_size query parameter: %w", err)
		}
		opts.PoolSize = poolSize
	}

	return opts, nil
}
