package mem

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	// Register the memory driver with the watcher framework.
	watcher.RegisterDriver("mem", &Driver{})
}

// Driver implements the watcher.Driver interface for the in-memory pub/sub.
// It manages the creation and lifecycle of PubSub instances.
type Driver struct {
	// once ensures that the shared pubsub instance is initialized only once.
	once sync.Once
	// pubsub is the shared memory-based pub/sub instance.
	pubsub *memoryPubSub
}

// memoryPubSub is a wrapper around gochannel.GoChannel to implement the watcher.PubSub interface.
type memoryPubSub struct {
	pubsub *gochannel.GoChannel
	shared bool
}

// Publish sends messages to the specified topic.
func (m *memoryPubSub) Publish(topic string, messages ...*message.Message) error {
	// gochannel.GoChannel's Publish can return an error if the pubsub is closed.
	return m.pubsub.Publish(topic, messages...)
}

// Subscribe returns a channel of messages for the specified topic.
func (m *memoryPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	// gochannel.GoChannel's Subscribe can return an error if the pubsub is closed.
	return m.pubsub.Subscribe(ctx, topic)
}

// Close shuts down the pub/sub.
func (m *memoryPubSub) Close() error {
	if m.shared {
		return nil
	}
	return m.pubsub.Close()
}

// NewPubSub creates a new PubSub instance based on the provided URL.
// It supports both shared and non-shared instances.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	query := u.Query()
	shared := true
	if s := query.Get("shared"); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, fmt.Errorf("invalid shared query parameter: %w", err)
		}
		shared = b
	}

	bufferSize := 0
	if bs := query.Get("buffer_size"); bs != "" {
		var err error
		bufferSize, err = strconv.Atoi(bs)
		if err != nil {
			return nil, fmt.Errorf("invalid buffer_size query parameter '%s': %w", bs, err)
		}
	}

	if shared {
		// Initialize the shared pubsub instance once.
		d.once.Do(func() {
			d.pubsub = &memoryPubSub{
				pubsub: gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: int64(bufferSize)}, logger),
				shared: true,
			}
		})
		return d.pubsub, nil
	}

	// For non-shared instances, create a new one.
	return &memoryPubSub{
		pubsub: gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: int64(bufferSize)}, logger),
		shared: false,
	}, nil
}
