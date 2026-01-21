package kafka

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("kafka", &Driver{})
}

type Driver struct{}

func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseKafkaURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka url: %w", err)
	}

	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               config.Brokers,
			Marshaler:             kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: config.SaramaConfig,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	subscriber, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               config.Brokers,
			Unmarshaler:           kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: config.SaramaConfig,
			ConsumerGroup:         config.ConsumerGroup,
		},
		logger,
	)
	if err != nil {
		if closeErr := publisher.Close(); closeErr != nil {
			logger.Error("failed to close Kafka publisher after subscriber creation failed", closeErr, nil)
		}
		return nil, err
	}

	return &kafkaPubSub{publisher: publisher, subscriber: subscriber}, nil
}

type kafkaPubSub struct {
	publisher  *kafka.Publisher
	subscriber *kafka.Subscriber
}

func (k *kafkaPubSub) Publish(topic string, messages ...*message.Message) error {
	return k.publisher.Publish(topic, messages...)
}

func (k *kafkaPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return k.subscriber.Subscribe(ctx, topic)
}

func (k *kafkaPubSub) Close() error {
	return multierr.Append(k.publisher.Close(), k.subscriber.Close())
}

type kafkaConfig struct {
	Brokers       []string
	ConsumerGroup string
	SaramaConfig  *sarama.Config
}

func parseKafkaURL(u *url.URL) (*kafkaConfig, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("kafka brokers are not specified in URL host")
	}

	config := &kafkaConfig{
		Brokers:      strings.Split(u.Host, ","),
		SaramaConfig: kafka.DefaultSaramaSubscriberConfig(),
	}

	query := u.Query()

	if cg := query.Get("consumer_group"); cg != "" {
		config.ConsumerGroup = cg
	}

	// TODO: Add more Sarama options from query parameters

	return config, nil
}
