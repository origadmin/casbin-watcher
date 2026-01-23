package nats

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	natsio "github.com/nats-io/nats.go"
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

	// --- Build Core NATS Options ---
	natsOptions := []natsio.Option{
		natsio.RetryOnFailedConnect(config.RetryOnFailedConnect),
	}
	if config.ConnectTimeout > 0 {
		natsOptions = append(natsOptions, natsio.Timeout(config.ConnectTimeout))
	}
	if config.ReconnectWait > 0 {
		natsOptions = append(natsOptions, natsio.ReconnectWait(config.ReconnectWait))
	}

	// --- Build Marshaler/Unmarshaler ---
	var marshalerUnmarshaler nats.MarshalerUnmarshaler
	switch config.Marshaler {
	case "json":
		marshalerUnmarshaler = &nats.JSONMarshaler{}
	default: // "gob"
		marshalerUnmarshaler = &nats.GobMarshaler{}
	}

	// --- Build JetStream Config ---
	jetStreamConfig := nats.JetStreamConfig{
		Disabled:      !config.JetStreamEnabled,
		AutoProvision: config.JetStreamEnabled && config.AutoProvision,
		TrackMsgId:    config.TrackMsgID,
		AckAsync:      config.AckAsync,
	}

	if config.JetStreamEnabled {
		jetStreamConfig.ConnectOptions = []natsio.JSOpt{
			natsio.PublishAsyncMaxPending(256),
		}

		subOpts := []natsio.SubOpt{natsio.AckExplicit()}
		switch config.DeliverPolicy {
		case "last":
			subOpts = append(subOpts, natsio.DeliverLast())
		case "new":
			subOpts = append(subOpts, natsio.DeliverNew())
		case "last_per_subject":
			subOpts = append(subOpts, natsio.DeliverLastPerSubject())
		default: // "all"
			subOpts = append(subOpts, natsio.DeliverAll())
		}

		if config.QueueGroup != "" {
			subOpts = append(subOpts, natsio.Durable(config.QueueGroup))
		}
		jetStreamConfig.SubscribeOptions = subOpts
	}

	// --- Create Publisher and Subscriber ---
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         config.Address,
			NatsOptions: natsOptions,
			Marshaler:   marshalerUnmarshaler,
			JetStream:   jetStreamConfig,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:              config.Address,
			CloseTimeout:     config.CloseTimeout,
			NatsOptions:      natsOptions,
			Unmarshaler:      marshalerUnmarshaler,
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
	Address              string
	QueueGroup           string
	JetStreamEnabled     bool
	AutoProvision        bool
	ConnectTimeout       time.Duration
	ReconnectWait        time.Duration
	CloseTimeout         time.Duration
	RetryOnFailedConnect bool
	DeliverPolicy        string
	AckAsync             bool
	TrackMsgID           bool
	Marshaler            string
}

func parseNatsURL(u *url.URL) (*natsConfig, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("nats address is not specified in URL host")
	}

	query := u.Query()
	config := &natsConfig{
		Address:              fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		QueueGroup:           query.Get("queue_group"),
		DeliverPolicy:        strings.ToLower(query.Get("deliver_policy")),
		Marshaler:            strings.ToLower(query.Get("marshaler")),
		RetryOnFailedConnect: true,             // Default to true for robustness
		AutoProvision:        true,             // Default to true for convenience, can be overridden.
		CloseTimeout:         30 * time.Second, // Default to 30s for graceful shutdown
	}

	var err error
	if js := query.Get("jetstream"); js != "" {
		config.JetStreamEnabled, err = strconv.ParseBool(js)
		if err != nil {
			return nil, fmt.Errorf("invalid 'jetstream' param: %w", err)
		}
	}
	if ap := query.Get("auto_provision"); ap != "" {
		config.AutoProvision, err = strconv.ParseBool(ap)
		if err != nil {
			return nil, fmt.Errorf("invalid 'auto_provision' param: %w", err)
		}
	}
	if ct := query.Get("connect_timeout"); ct != "" {
		config.ConnectTimeout, err = time.ParseDuration(ct)
		if err != nil {
			return nil, fmt.Errorf("invalid 'connect_timeout' param: %w", err)
		}
	}
	if rw := query.Get("reconnect_wait"); rw != "" {
		config.ReconnectWait, err = time.ParseDuration(rw)
		if err != nil {
			return nil, fmt.Errorf("invalid 'reconnect_wait' param: %w", err)
		}
	}
	if ct := query.Get("close_timeout"); ct != "" {
		config.CloseTimeout, err = time.ParseDuration(ct)
		if err != nil {
			return nil, fmt.Errorf("invalid 'close_timeout' param: %w", err)
		}
	}
	if rofc := query.Get("retry_on_failed_connect"); rofc != "" {
		config.RetryOnFailedConnect, err = strconv.ParseBool(rofc)
		if err != nil {
			return nil, fmt.Errorf("invalid 'retry_on_failed_connect' param: %w", err)
		}
	}
	if aa := query.Get("ack_async"); aa != "" {
		config.AckAsync, err = strconv.ParseBool(aa)
		if err != nil {
			return nil, fmt.Errorf("invalid 'ack_async' param: %w", err)
		}
	}
	if tmi := query.Get("track_msg_id"); tmi != "" {
		config.TrackMsgID, err = strconv.ParseBool(tmi)
		if err != nil {
			return nil, fmt.Errorf("invalid 'track_msg_id' param: %w", err)
		}
	}

	return config, nil
}
