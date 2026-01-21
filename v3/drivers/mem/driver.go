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
	globalMemoryPubSubMu sync.Mutex
)

type memoryPubSub struct {
	pubsub *gochannel.GoChannel
}

func (m *memoryPubSub) Publish(topic string, messages ...*message.Message) error {
	return m.pubsub.Publish(topic, messages...)
}

func (m *memoryPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return m.pubsub.Subscribe(ctx, topic)
}

func (m *memoryPubSub) Close() error {
	return m.pubsub.Close()
}

type memoryPubSubWrapper struct {
	watcher.PubSub
	shared bool
}

func (w *memoryPubSubWrapper) Close() error {
	if !w.shared {
		return w.PubSub.Close()
	}
	return nil
}

func (d *Driver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
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
		globalMemoryPubSubMu.Lock()
		defer globalMemoryPubSubMu.Unlock()
		if globalMemoryPubSub == nil {
			globalMemoryPubSub = &memoryPubSub{
				pubsub: gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: int64(bufferSize)}, logger),
			}
		}
		return &memoryPubSubWrapper{PubSub: globalMemoryPubSub, shared: true}, nil
	} else {
		ps := &memoryPubSub{
			pubsub: gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: int64(bufferSize)}, logger),
		}
		return &memoryPubSubWrapper{PubSub: ps, shared: false}, nil
	}
}
