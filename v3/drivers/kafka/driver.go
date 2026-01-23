package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	watcher "github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("kafka", &Driver{})
}

// Driver for Kafka.
type Driver struct{}

// NewPubSub creates a new Pub/Sub instance for Kafka.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseKafkaURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka url: %w", err)
	}

	var marshalerUnmarshaler kafka.MarshalerUnmarshaler
	switch config.Marshaler {
	case "json":
		marshalerUnmarshaler = &JSONMarshaler{} // Use custom JSONMarshaler
	default: // "default"
		marshalerUnmarshaler = kafka.DefaultMarshaler{}
	}

	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               config.Brokers,
			Marshaler:             marshalerUnmarshaler,
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
			Unmarshaler:           marshalerUnmarshaler, // Use the same marshaler for unmarshaling
			OverwriteSaramaConfig: config.SaramaConfig,
			ConsumerGroup:         config.ConsumerGroup,
			// CloseTimeout is not a field in watermill-kafka's SubscriberConfig.
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
	Marshaler     string
	InitialOffset string
	// CloseTimeout is not supported by watermill-kafka's SubscriberConfig.
}

// JSONMarshaler is a simple JSON marshaler for Kafka.
type JSONMarshaler struct{}

// Marshal implements Marshaler interface.
func (j JSONMarshaler) Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error) {
	b, err := json.Marshal(msg.Payload)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(b),
	}, nil
}

// Unmarshal implements Unmarshaler interface.
func (j JSONMarshaler) Unmarshal(kafkaMsg *sarama.ConsumerMessage) (*message.Message, error) {
	msg := message.NewMessage(watermill.NewUUID(), kafkaMsg.Value)
	msg.Metadata.Set("kafka_topic", kafkaMsg.Topic)
	msg.Metadata.Set("kafka_partition", fmt.Sprintf("%d", kafkaMsg.Partition))
	msg.Metadata.Set("kafka_offset", fmt.Sprintf("%d", kafkaMsg.Offset))
	return msg, nil
}

func parseKafkaURL(u *url.URL) (*kafkaConfig, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("kafka brokers are not specified in URL host")
	}

	config := &kafkaConfig{
		Brokers:       strings.Split(u.Host, ","),
		SaramaConfig:  kafka.DefaultSaramaSubscriberConfig(), // v3 should return IBM Sarama config
		ConsumerGroup: "casbin-watcher",                      // Robust default
		Marshaler:     "default",                             // Default marshaler
		InitialOffset: "newest",                              // Default initial offset
		// CloseTimeout is not supported by watermill-kafka's SubscriberConfig.
	}

	query := u.Query()

	if cg := query.Get("consumer_group"); cg != "" {
		config.ConsumerGroup = cg
	}

	if m := query.Get("marshaler"); m != "" {
		config.Marshaler = m
	}

	if io := query.Get("initial_offset"); io != "" {
		config.InitialOffset = io
	}

	// CloseTimeout is not supported by watermill-kafka's SubscriberConfig.
	// if ct := query.Get("close_timeout"); ct != "" {
	// 	val, err := time.ParseDuration(ct)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("invalid 'close_timeout' param: %w", err)
	// 	}
	// 	config.CloseTimeout = val
	// }

	// --- Sarama Config Overrides ---
	// Initial Offset
	switch strings.ToLower(config.InitialOffset) {
	case "oldest":
		config.SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest":
		config.SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		return nil, fmt.Errorf("invalid 'initial_offset' param: %s, must be 'oldest' or 'newest'", config.InitialOffset)
	}

	// SASL Authentication
	if saslEnable := query.Get("sasl_enable"); strings.ToLower(saslEnable) == "true" {
		username := query.Get("sasl_user")
		password := query.Get("sasl_password")

		if username != "" && password != "" {
			config.SaramaConfig.Net.SASL.Enable = true
			config.SaramaConfig.Net.SASL.User = username
			config.SaramaConfig.Net.SASL.Password = password
			config.SaramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext // Only Plaintext is supported for now
		} else {
			return nil, fmt.Errorf("sasl_enable is true, but sasl_user or sasl_password is missing")
		}
	}

	return config, nil
}
