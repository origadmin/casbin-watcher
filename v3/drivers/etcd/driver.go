package etcd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.etcd.io/etcd/client/v3"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("etcd", &Driver{})
}

// Driver implements the watcher.Driver interface for etcd.
type Driver struct{}

// NewPubSub creates a new etcd PubSub instance.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseEtcdURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd url: %w", err)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
		Username:    config.User,
		Password:    config.Password,
	})
	if err != nil {
		return nil, err
	}

	// Both publisher and subscriber will share the same client.
	// The client will be closed by the pubSub wrapper.
	publisher := &Publisher{cli: cli, logger: logger}
	subscriber := &Subscriber{cli: cli, logger: logger}

	return &pubSub{
		cli:        cli,
		publisher:  publisher,
		subscriber: subscriber,
	}, nil
}

// pubSub is a wrapper that manages the lifecycle of the shared etcd client.
type pubSub struct {
	cli        *clientv3.Client
	publisher  *Publisher
	subscriber *Subscriber
	closeOnce  sync.Once
}

// Publish publishes messages to etcd.
func (e *pubSub) Publish(topic string, messages ...*message.Message) error {
	return e.publisher.Publish(topic, messages...)
}

// Subscribe subscribes to a topic in etcd.
func (e *pubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return e.subscriber.Subscribe(ctx, topic)
}

// Close ensures the underlying etcd client is closed only once.
func (e *pubSub) Close() error {
	var err error
	e.closeOnce.Do(func() {
		err = e.cli.Close()
	})
	return err
}

// Publisher implements message.Publisher for etcd.
type Publisher struct {
	cli    *clientv3.Client
	logger watermill.LoggerAdapter
}

// Publish publishes messages to etcd.
func (p *Publisher) Publish(topic string, messages ...*message.Message) error {
	var allErrors error
	for _, msg := range messages {
		// Use a key structure that is easy to watch with a prefix.
		key := topic + "/" + msg.UUID
		_, err := p.cli.Put(context.Background(), key, string(msg.Payload))
		if err != nil {
			p.logger.Error("Failed to publish message to etcd", err, watermill.LogFields{"topic": topic, "key": key})
			allErrors = multierr.Append(allErrors, err)
		}
	}
	return allErrors
}

// Close is a no-op because the client is managed by the pubSub wrapper.
func (p *Publisher) Close() error {
	return nil
}

// Subscriber implements message.Subscriber for etcd.
type Subscriber struct {
	cli    *clientv3.Client
	logger watermill.LoggerAdapter
}

// Subscribe subscribes to a topic in etcd.
func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	output := make(chan *message.Message)

	go func() {
		defer close(output)

		// Start watching from the current revision to avoid getting old messages.
		watchChan := s.cli.Watch(ctx, topic, clientv3.WithPrefix())

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("Etcd subscriber context done, stopping watch.", watermill.LogFields{"topic": topic})
				return
			case watchResp, ok := <-watchChan:
				if !ok {
					s.logger.Error("Etcd watch channel closed.", nil, watermill.LogFields{"topic": topic})
					return
				}
				if err := watchResp.Err(); err != nil {
					s.logger.Error("Etcd watch error.", err, watermill.LogFields{"topic": topic})
					// The watch might be compacted or cancelled. The consumer should handle reconnection if needed.
					return
				}

				for _, event := range watchResp.Events {
					// We only care about new values being put.
					if event.Type == clientv3.EventTypePut {
						// Create a new message and send it to the output channel.
						// The UUID is newly generated as we are creating a new watermill message.
						msg := message.NewMessage(watermill.NewUUID(), event.Kv.Value)
						output <- msg
					}
				}
			}
		}
	}()

	return output, nil
}

// Close is a no-op because the client is managed by the pubSub wrapper.
func (s *Subscriber) Close() error {
	return nil
}

type etcdConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	User        string
	Password    string
}

func parseEtcdURL(u *url.URL) (*etcdConfig, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("etcd endpoints are not specified in URL host")
	}

	query := u.Query()
	config := &etcdConfig{
		Endpoints:   strings.Split(u.Host, ","),
		DialTimeout: 5 * time.Second, // Default dial timeout
	}

	if dt := query.Get("dial_timeout"); dt != "" {
		val, err := time.ParseDuration(dt)
		if err != nil {
			return nil, fmt.Errorf("invalid 'dial_timeout' param: %w", err)
		}
		config.DialTimeout = val
	}

	if user := u.User.Username(); user != "" {
		config.User = user
		if pass, ok := u.User.Password(); ok {
			config.Password = pass
		}
	}

	return config, nil
}
