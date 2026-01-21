package rabbitmq

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("amqp", &Driver{})
}

type Driver struct{}

func (d *Driver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: u.String(),
		},
	}

	publisher, err := amqp.NewPublisher(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create amqp publisher: %w", err)
	}

	subscriber, err := amqp.NewSubscriber(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create amqp subscriber: %w", err)
	}

	return &amqpPubSub{publisher: publisher, subscriber: subscriber}, nil
}

type amqpPubSub struct {
	publisher  *amqp.Publisher
	subscriber *amqp.Subscriber
}

func (a *amqpPubSub) Publish(topic string, messages ...*message.Message) error {
	return a.publisher.Publish(topic, messages...)
}

func (a *amqpPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return a.subscriber.Subscribe(ctx, topic)
}

func (a *amqpPubSub) Close() error {
	errPub := a.publisher.Close()
	errSub := a.subscriber.Close()
	if errPub != nil {
		return errPub
	}
	return errSub
}
