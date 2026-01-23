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
	watcher.RegisterDriver("mem", &Driver{})
}

type Driver struct{}

// globalMemoryPubSub ensures that all memory-based watchers in the same process
// can communicate with each other.
var (
	globalMemoryPubSub   *memoryPubSub
	globalMemoryPubSubMu sync.RWMutex // Use RWMutex for better concurrency control
)

type memoryPubSub struct {
	pubsub *gochannel.GoChannel
}

func (m *memoryPubSub) Publish(topic string, messages ...*message.Message) error {
	// gochannel.GoChannel's Publish can return an error if the pubsub is closed.
	return m.pubsub.Publish(topic, messages...)
}

func (m *memoryPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	// gochannel.GoChannel's Subscribe can return an error if the pubsub is closed.
	return m.pubsub.Subscribe(ctx, topic)
}

func (m *memoryPubSub) Close() error {
	return m.pubsub.Close()
}

// memoryPubSubWrapper wraps the actual PubSub to handle shared instance closing logic.
type memoryPubSubWrapper struct {
	watcher.PubSub
	shared bool
}

// Close for the wrapper. If the underlying PubSub is shared, it's a no-op.
func (w *memoryPubSubWrapper) Close() error {
	if !w.shared {
		return w.PubSub.Close()
	}
	return nil
}

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
		globalMemoryPubSubMu.Lock() // Write lock for creating the global instance
		defer globalMemoryPubSubMu.Unlock()
		if globalMemoryPubSub == nil {
			globalMemoryPubSub = &memoryPubSub{
				pubsub: gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: int64(bufferSize)}, logger),
			}
		}
		// For shared instances, we need to ensure the global instance is not nil before returning.
		// A read lock is not strictly needed here as the write lock is held, but conceptually,
		- // subsequent NewPubSub calls will acquire a read lock to access globalMemoryPubSub.
		return &memoryPubSubWrapper{PubSub: globalMemoryPubSub, shared: true}, nil
	}

	// For non-shared instances, create a new one.
	ps := &memoryPubSub{
		pubsub: gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: int64(bufferSize)}, logger),
	}
	return &memoryPubSubWrapper{PubSub: ps, shared: false}, nil
}
