package nats

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("nats", &Driver{})
}

type Driver struct{}

func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseNatsURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nats url: %w", err)
	}

	jetStreamConfig := nats.JetStreamConfig{
		Disabled: !config.JetStreamEnabled,
	}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:       config.Address,
			JetStream: jetStreamConfig,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:              config.Address,
			QueueGroupPrefix: config.QueueGroup,
			JetStream:        jetStreamConfig,
		},
		logger,
	)
	if err != nil {
		if closeErr := publisher.Close(); closeErr != nil {
			logger.Error("failed to close NATS publisher after subscriber creation failed", closeErr, nil)
		}
		return nil, err
	}

	return &natsPubSub{publisher: publisher, subscriber: subscriber}, nil
}

type natsPubSub struct {
	publisher  *nats.Publisher
	subscriber *nats.Subscriber
}

func (n *natsPubSub) Publish(topic string, messages ...*message.Message) error {
	return n.publisher.Publish(topic, messages...)
}

func (n *natsPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return n.subscriber.Subscribe(ctx, topic)
}

func (n *natsPubSub) Close() error {
	return multierr.Append(n.publisher.Close(), n.subscriber.Close())
}

type natsConfig struct {
	Address          string
	QueueGroup       string
	JetStreamEnabled bool
}

func parseNatsURL(u *url.URL) (*natsConfig, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("nats address is not specified in URL host")
	}

	config := &natsConfig{
		Address: fmt.Sprintf("%s://%s", u.Scheme, u.Host),
	}

	query := u.Query()

	if qg := query.Get("queue_group"); qg != "" {
		config.QueueGroup = qg
	}
	if js := query.Get("jetstream"); js != "" {
		var err error
		config.JetStreamEnabled, err = strconv.ParseBool(js)
		if err != nil {
			return nil, fmt.Errorf("invalid jetstream query parameter '%s': %w", js, err)
		}
	}

	return config, nil
}
