package gcpv2

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("gcpv2", &Driver{})
}

// Driver for Google Cloud Pub/Sub.
type Driver struct{}

// NewPubSub creates a new Pub/Sub instance for Google Cloud Pub/Sub.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseGcpURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gcp url: %w", err)
	}

	publisher, err := googlecloud.NewPublisher(
		googlecloud.PublisherConfig{
			ProjectID: config.ProjectID,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP publisher: %w", err)
	}

	subscriber, err := googlecloud.NewSubscriber(
		googlecloud.SubscriberConfig{
			ProjectID: config.ProjectID,
			// GenerateSubscriptionName will use the topic name as the subscription name by default.
			Unmarshaler: googlecloud.DefaultMarshalerUnmarshaler{},
		},
		logger,
	)
	if err != nil {
		if closeErr := publisher.Close(); closeErr != nil {
			logger.Error("failed to close GCP publisher after subscriber creation failed", closeErr, nil)
		}
		return nil, fmt.Errorf("failed to create GCP subscriber: %w", err)
	}

	return &gcpPubSub{
		publisher:  publisher,
		subscriber: subscriber,
	}, nil
}

type gcpPubSub struct {
	publisher  *googlecloud.Publisher
	subscriber *googlecloud.Subscriber // Corrected type
}

func (g *gcpPubSub) Publish(topic string, messages ...*message.Message) error {
	return g.publisher.Publish(topic, messages...)
}

func (g *gcpPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return g.subscriber.Subscribe(ctx, topic)
}

func (g *gcpPubSub) Close() error {
	return multierr.Append(g.publisher.Close(), g.subscriber.Close())
}

type gcpConfig struct {
	ProjectID string
	// SubscriptionID is not directly used in the config, as GenerateSubscriptionName handles it.
	// Keeping it in parseGcpURL for potential future use or custom generation.
}

func parseGcpURL(u *url.URL) (*gcpConfig, error) {
	config := &gcpConfig{}

	if u.Host == "" {
		return nil, fmt.Errorf("gcp project id is not specified in URL host")
	}
	config.ProjectID = u.Host

	// The topic is not needed for configuration, it's passed directly to Publish/Subscribe methods.
	// The subscription ID is handled by GenerateSubscriptionName in SubscriberConfig.
	// If a custom subscription name is needed, it can be parsed here and passed to GenerateSubscriptionName.

	return config, nil
}
