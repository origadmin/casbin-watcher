package watcher_test

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/origadmin/casbin-watcher/v3"
)

// Test context key for passing custom configuration
type contextKey string

const testConfigKey contextKey = "testConfig"

func TestWatcherWithCustomDriver(t *testing.T) {
	// Register a custom driver that uses context configuration
	watcher.RegisterDriver("test-ctx", &testDriver{})

	tests := []struct {
		name             string
		ctx              context.Context
		expectedMessage  string
		expectedBehavior string
	}{
		{
			name:             "Without custom config - uses default behavior",
			ctx:              context.Background(),
			expectedMessage:  "default-update",
			expectedBehavior: "Driver should use default message when no config provided",
		},
		{
			name:             "With custom config - uses configured behavior",
			ctx:              context.WithValue(context.Background(), testConfigKey, "custom"),
			expectedMessage:  "custom-update",
			expectedBehavior: "Driver should use custom behavior from context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testURL := "test-ctx://casbin?shared=true"
			updateCh := make(chan string, 1)

			updater, err := watcher.NewWatcher(tt.ctx, testURL)
			require.NoError(t, err)
			defer updater.Close()

			listener, err := watcher.NewWatcher(tt.ctx, testURL)
			require.NoError(t, err)
			defer listener.Close()

			err = listener.SetUpdateCallback(func(msg string) {
				updateCh <- msg
			})
			require.NoError(t, err)

			err = updater.Update()
			require.NoError(t, err)

			select {
			case msg := <-updateCh:
				// Message content differs based on context configuration
				require.Equal(t, tt.expectedMessage, msg, tt.expectedBehavior)
			case <-time.After(time.Second * 5):
				t.Fatal("Listener didn't receive message in time")
			}
		})
	}
}

// testDriver is a mock driver that uses context configuration to determine behavior
type testDriver struct {
	instances map[string]*contextAwarePubSub // map of URL to shared PubSub instance
	mu        sync.Mutex
}

func (d *testDriver) NewPubSub(ctx context.Context, parsedURL *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.instances == nil {
		d.instances = make(map[string]*contextAwarePubSub)
	}

	key := parsedURL.String()

	// Check if custom configuration is present in context
	// This config determines the behavior of the watcher
	var messagePrefix string
	if config, ok := ctx.Value(testConfigKey).(string); ok {
		messagePrefix = config
	} else {
		messagePrefix = "default"
	}

	// Check for shared instance
	if ps, ok := d.instances[key]; ok {
		// Update the message prefix for the existing instance to reflect the new context
		ps.messagePrefix = messagePrefix
		return &pubSubWrapper{PubSub: ps, shared: true}, nil
	}

	// Create a new shared instance
	ps := &contextAwarePubSub{
		messagePrefix: messagePrefix,
		logger:        logger,
		subscribers:   make([]chan *message.Message, 0),
	}
	d.instances[key] = ps
	return &pubSubWrapper{PubSub: ps, shared: true}, nil
}

// pubSubWrapper wraps PubSub to handle shared instance closing logic
type pubSubWrapper struct {
	watcher.PubSub
	shared bool
}

func (w *pubSubWrapper) Close() error {
	if !w.shared {
		return w.PubSub.Close()
	}
	return nil
}

// contextAwarePubSub demonstrates behavior changes based on context configuration
type contextAwarePubSub struct {
	messagePrefix string
	logger        watermill.LoggerAdapter
	subscribers   []chan *message.Message
	mu            sync.Mutex
	closed        bool
}

func (c *contextAwarePubSub) Publish(topic string, messages ...*message.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("pubsub is closed")
	}

	// Modify message payload based on context configuration
	for _, msg := range messages {
		msg.Payload = []byte(c.messagePrefix + "-update")
		// Send to all subscribers
		for _, sub := range c.subscribers {
			select {
			case sub <- msg:
			default:
				// Channel full, skip
			}
		}
	}
	return nil
}

func (c *contextAwarePubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message, 10)
	c.mu.Lock()
	c.subscribers = append(c.subscribers, ch)
	c.mu.Unlock()
	return ch, nil
}

func (c *contextAwarePubSub) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for _, sub := range c.subscribers {
		close(sub)
	}
	return nil
}
