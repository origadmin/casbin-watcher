package aws

import (
	"context"
	"fmt"
	"net/url"

	"go.uber.org/multierr"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	// Register the SNS+SQS driver.
	watcher.RegisterDriver("snssqs", &SnsSqsDriver{})
}

// SnsSqsDriver implements a combined SNS publisher and SQS subscriber.
type SnsSqsDriver struct{}

// NewPubSub creates a new Pub/Sub instance using SNS for publishing and SQS for subscribing.
func (d *SnsSqsDriver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseSnsSqsURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse snssqs url: %w", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	publisher, err := sns.NewPublisher(
		sns.PublisherConfig{
			AWSConfig: awsCfg,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNS publisher: %w", err)
	}

	subscriber, err := sqs.NewSubscriber(
		sqs.SubscriberConfig{
			AWSConfig: awsCfg,
		},
		logger,
	)
	if err != nil {
		if closeErr := publisher.Close(); closeErr != nil {
			logger.Error("failed to close SNS publisher after subscriber creation failed", closeErr, nil)
		}
		return nil, fmt.Errorf("failed to create SQS subscriber: %w", err)
	}

	return &snsSqsPubSub{
		publisher:  publisher,
		subscriber: subscriber,
		topicARN:   config.TopicARN,
		queueURL:   config.QueueURL,
	}, nil
}

type snsSqsPubSub struct {
	publisher  *sns.Publisher
	subscriber *sqs.Subscriber
	topicARN   string
	queueURL   string
}

func (s *snsSqsPubSub) Publish(_ string, messages ...*message.Message) error {
	return s.publisher.Publish(s.topicARN, messages...)
}

func (s *snsSqsPubSub) Subscribe(ctx context.Context, _ string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, s.queueURL)
}

func (s *snsSqsPubSub) Close() error {
	return multierr.Append(s.publisher.Close(), s.subscriber.Close())
}

type snsSqsConfig struct {
	Region   string
	QueueURL string
	TopicARN string
}

func parseSnsSqsURL(u *url.URL) (*snsSqsConfig, error) {
	config := &snsSqsConfig{}

	if region := u.Query().Get("region"); region != "" {
		config.Region = region
	} else {
		return nil, fmt.Errorf("aws region is not specified in URL query 'region'")
	}

	if topicArn := u.Query().Get("topic_arn"); topicArn != "" {
		config.TopicARN = topicArn
	} else {
		return nil, fmt.Errorf("sns topic arn is not specified in URL query 'topic_arn'")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("sqs queue url is not specified in URL host")
	}
	config.QueueURL = "https://" + u.Host + u.Path

	return config, nil
}
