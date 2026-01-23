package io

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	watermillio "github.com/ThreeDotsLabs/watermill-io/pkg/io"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/multierr"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("io", &Driver{})
}

// Driver implements the watcher.Driver interface for IO.
type Driver struct{}

// NewPubSub creates a new PubSub for IO.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	var reader io.ReadCloser = os.Stdin
	var writer io.WriteCloser = os.Stdout
	var closer io.Closer

	path := u.Query().Get("path")
	if path != "" {
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for io driver: %w", err)
		}
		reader = file
		writer = file
		closer = file
	}

	publisher, err := watermillio.NewPublisher(writer, watermillio.PublisherConfig{}, logger)
	if err != nil {
		return nil, err
	}

	subscriber, err := watermillio.NewSubscriber(reader, watermillio.SubscriberConfig{
		UnmarshalFunc: watermillio.PayloadUnmarshalFunc,
	}, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, err
	}

	return &pubSub{
		publisher:  publisher,
		subscriber: subscriber,
		closer:     closer,
	}, nil
}

type pubSub struct {
	publisher  *watermillio.Publisher
	subscriber *watermillio.Subscriber
	closer     io.Closer
}

func (p *pubSub) Publish(topic string, messages ...*message.Message) error {
	// The IO publisher doesn't use topics, it just writes to the writer.
	return p.publisher.Publish(topic, messages...)
}

func (p *pubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	// The IO subscriber doesn't use topics, it just reads from the reader.
	return p.subscriber.Subscribe(ctx, topic)
}

func (p *pubSub) Close() error {
	var allErrors error
	allErrors = multierr.Append(allErrors, p.publisher.Close())
	allErrors = multierr.Append(allErrors, p.subscriber.Close())
	if p.closer != nil {
		allErrors = multierr.Append(allErrors, p.closer.Close())
	}
	return allErrors
}
