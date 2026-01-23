package aws

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/multierr"

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

	var snsMarshaler sns.Marshaler
	var sqsUnmarshaler sqs.Unmarshaler

	switch config.Marshaler {
	case "json":
		snsMarshaler = &sns.JSONMarshaler{}
		sqsUnmarshaler = &sqs.JSONMarshaler{}
	default: // "default"
		snsMarshaler = &sns.DefaultMarshaler{}
		sqsUnmarshaler = &sqs.DefaultMarshaler{}
	}

	publisher, err := sns.NewPublisher(
		sns.PublisherConfig{
			AWSConfig: awsCfg,
			Marshaler: snsMarshaler,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SNS publisher: %w", err)
	}

	subscriber, err := sqs.NewSubscriber(
		sqs.SubscriberConfig{
			AWSConfig:    awsCfg,
			Unmarshaler:  sqsUnmarshaler,
			CloseTimeout: config.CloseTimeout,
			ReceiveMessageParams: &types.ReceiveMessageInput{
				WaitTimeSeconds:       int32(config.WaitTimeSeconds),
				VisibilityTimeout:     int32(config.VisibilityTimeout),
				MaxNumberOfMessages:   10, // A reasonable default for batching
				MessageAttributeNames: []string{"All"},
			},
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
	// In this pattern, the topic is fixed to the configured SNS Topic ARN.
	return s.publisher.Publish(s.topicARN, messages...)
}

func (s *snsSqsPubSub) Subscribe(ctx context.Context, _ string) (<-chan *message.Message, error) {
	// The subscription is always from the configured SQS Queue URL.
	return s.subscriber.Subscribe(ctx, s.queueURL)
}

func (s *snsSqsPubSub) Close() error {
	return multierr.Append(s.publisher.Close(), s.subscriber.Close())
}

type snsSqsConfig struct {
	Region            string
	QueueURL          string
	TopicARN          string
	Marshaler         string
	WaitTimeSeconds   int
	VisibilityTimeout int
	CloseTimeout      time.Duration
}

func parseSnsSqsURL(u *url.URL) (*snsSqsConfig, error) {
	config := &snsSqsConfig{
		// Set robust defaults
		Marshaler:         "default",
		WaitTimeSeconds:   20, // Enable long polling by default
		VisibilityTimeout: 30, // Default visibility timeout
		CloseTimeout:      30 * time.Second,
	}

	if region := u.Query().Get("region"); region != "" {
		config.Region = region
	} else {
		return nil, fmt.Errorf("aws region is not specified in URL query 'region'")
	}

	// For SNS+SQS, the host/path is the SQS Queue Name, and topic_arn is a query param.
	if u.Host == "" {
		return nil, fmt.Errorf("sqs queue name is not specified in URL host")
	}
	config.QueueURL = "https://" + u.Host + u.Path

	if topicArn := u.Query().Get("topic_arn"); topicArn != "" {
		config.TopicARN = topicArn
	} else {
		return nil, fmt.Errorf("sns topic arn is not specified in URL query 'topic_arn'")
	}

	if m := u.Query().Get("marshaler"); m != "" {
		config.Marshaler = m
	}
	if wt := u.Query().Get("wait_time_seconds"); wt != "" {
		val, err := strconv.Atoi(wt)
		if err != nil {
			return nil, fmt.Errorf("invalid 'wait_time_seconds' param: %w", err)
		}
		config.WaitTimeSeconds = val
	}
	if vt := u.Query().Get("visibility_timeout"); vt != "" {
		val, err := strconv.Atoi(vt)
		if err != nil {
			return nil, fmt.Errorf("invalid 'visibility_timeout' param: %w", err)
		}
		config.VisibilityTimeout = val
	}
	if ct := u.Query().Get("close_timeout"); ct != "" {
		val, err := time.ParseDuration(ct)
		if err != nil {
			return nil, fmt.Errorf("invalid 'close_timeout' param: %w", err)
		}
		config.CloseTimeout = val
	}

	return config, nil
}
