package firestore

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-firestore/pkg/firestore"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("firestore", &Driver{})
}

// Driver implements the watcher.Driver interface for Google Cloud Firestore.
type Driver struct{}

// NewPubSub creates a new PubSub for Firestore.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	projectID := u.Query().Get("project_id")
	if projectID == "" {
		return nil, fmt.Errorf("firestore driver requires a 'project_id' query parameter")
	}

	// The subscription ID is derived from the URL query parameter.
	subscriptionID := u.Query().Get("subscription_id")

	publisher, err := firestore.NewPublisher(firestore.PublisherConfig{
		ProjectID: projectID,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore publisher: %w", err)
	}

	subscriberConfig := firestore.SubscriberConfig{
		ProjectID: projectID,
	}

	if subscriptionID != "" {
		subscriberConfig.GenerateSubscriptionName = func(topic string) string {
			return subscriptionID
		}
	} else {
		// Generate a unique subscription ID if not provided.
		subscriberConfig.GenerateSubscriptionName = func(topic string) string {
			return fmt.Sprintf("%s-%s", topic, watermill.NewUUID())
		}
	}

	subscriber, err := firestore.NewSubscriber(subscriberConfig, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("failed to create firestore subscriber: %w", err)
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
	}, nil
}

type pubSub struct {
	publisher  *firestore.Publisher
	subscriber *firestore.Subscriber
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
	return allErrors
}
