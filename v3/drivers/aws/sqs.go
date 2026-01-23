package aws

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	// Register the SQS-only driver.
	watcher.RegisterDriver("sqs", &SQSDriver{})
}

// SQSDriver implements a pure SQS publisher and subscriber.
type SQSDriver struct{}

// NewPubSub creates a new Pub/Sub instance using SQS for both publishing and subscribing.
func (d *SQSDriver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseSqsOnlyURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sqs url: %w", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var marshalerUnmarshaler sqs.MarshalerUnmarshaler
	switch config.Marshaler {
	case "json":
		marshalerUnmarshaler = &sqs.JSONMarshaler{}
	default: // "default"
		marshalerUnmarshaler = &sqs.DefaultMarshaler{}
	}

	publisher, err := sqs.NewPublisher(
		sqs.PublisherConfig{
			AWSConfig: awsCfg,
			Marshaler: marshalerUnmarshaler,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQS publisher: %w", err)
	}

	subscriber, err := sqs.NewSubscriber(
		sqs.SubscriberConfig{
			AWSConfig:    awsCfg,
			Unmarshaler:  marshalerUnmarshaler,
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
			logger.Error("failed to close SQS publisher after subscriber creation failed", closeErr, nil)
		}
		return nil, fmt.Errorf("failed to create SQS subscriber: %w", err)
	}

	return &sqsOnlyPubSub{
		publisher:  publisher,
		subscriber: subscriber,
		queueURL:   config.QueueURL,
	}, nil
}

type sqsOnlyPubSub struct {
	publisher  *sqs.Publisher
	subscriber *sqs.Subscriber
	queueURL   string
}

func (s *sqsOnlyPubSub) Publish(_ string, messages ...*message.Message) error {
	return s.publisher.Publish(s.queueURL, messages...)
}

func (s *sqsOnlyPubSub) Subscribe(ctx context.Context, _ string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, s.queueURL)
}

func (s *sqsOnlyPubSub) Close() error {
	return multierr.Append(s.publisher.Close(), s.subscriber.Close())
}

type sqsOnlyConfig struct {
	Region            string
	QueueURL          string
	Marshaler         string
	WaitTimeSeconds   int
	VisibilityTimeout int
	CloseTimeout      time.Duration
}

func parseSqsOnlyURL(u *url.URL) (*sqsOnlyConfig, error) {
	config := &sqsOnlyConfig{
		// Set robust defaults
		Marshaler:         "default",
		WaitTimeSeconds:   20, // Enable long polling by default
		VisibilityTimeout: 30, // Default visibility timeout
		CloseTimeout:      30 * time.Second,
	}

	if region := u.Query().Get("region"); region != "" {
		config.Region = region
	} else {
		return nil, fmt.Errorf("sqs region is not specified in URL query 'region'")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("sqs queue url is not specified in URL host")
	}
	config.QueueURL = "https://" + u.Host + u.Path

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
