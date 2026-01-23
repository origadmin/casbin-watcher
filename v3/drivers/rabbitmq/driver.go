package rabbitmq

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	// Register the driver under the name "rabbitmq".
	// While the underlying library is for AMQP, users often associate it with RabbitMQ.
	watcher.RegisterDriver("rabbitmq", &Driver{})
}

// Driver implements the watcher.Driver interface for RabbitMQ.
type Driver struct{}

// NewPubSub creates a new PubSub for RabbitMQ.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	// Use a durable queue configuration to ensure messages are not lost on broker restart.
	// The topic from the watcher will be used as the queue name.
	config := amqp.NewDurableQueueConfig(u.String())

	publisher, err := amqp.NewPublisher(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create amqp publisher: %w", err)
	}

	subscriber, err := amqp.NewSubscriber(config, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("failed to create amqp subscriber: %w", err)
	}

	return &pubSub{publisher: publisher, subscriber: subscriber}, nil
}

type pubSub struct {
	publisher  *amqp.Publisher
	subscriber *amqp.Subscriber
}

func (a *pubSub) Publish(topic string, messages ...*message.Message) error {
	return a.publisher.Publish(topic, messages...)
}

func (a *pubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return a.subscriber.Subscribe(ctx, topic)
}

func (a *pubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, a.publisher.Close())
	allErrors = multierr.Append(allErrors, a.subscriber.Close())
	return allErrors
}
