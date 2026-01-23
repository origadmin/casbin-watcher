package http

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-http/v2/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("http", &Driver{})
}

// Driver implements the watcher.Driver interface for HTTP.
type Driver struct{}

// NewPubSub creates a new PubSub for HTTP.
// Note: The HTTP driver in this context only supports publishing.
// The subscriber part would require running a server, which is outside the scope of this driver.
func (d *Driver) NewPubSub(_ context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	baseURL := u.Query().Get("base_url")
	if baseURL == "" {
		return nil, fmt.Errorf("http driver requires a 'base_url' query parameter")
	}

	publisher, err := http.NewPublisher(
		http.PublisherConfig{}, // An empty config is sufficient to use the default HTTP client.
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http publisher: %w", err)
	}

	return &pubSub{
		publisher: publisher,
		baseURL:   baseURL,
	}, nil
}

type pubSub struct {
	publisher *http.Publisher
	baseURL   string
}

// Publish sends the message to a URL derived from the base_url and topic.
// For example, with base_url="http://localhost:8080" and topic="casbin", it will POST to "http://localhost:8080/casbin".
func (p *pubSub) Publish(topic string, messages ...*message.Message) error {
	// The watermill-http publisher's "topic" is the full URL.
	publishURL := fmt.Sprintf("%s/%s", p.baseURL, topic)
	return p.publisher.Publish(publishURL, messages...)
}

// Subscribe is not implemented for this HTTP driver.
func (p *pubSub) Subscribe(_ context.Context, _ string) (<-chan *message.Message, error) {
	return nil, fmt.Errorf("subscribing is not supported in this simple HTTP driver; " +
		"you need to set up an HTTP server with watermill's http.Router to receive messages")
}

func (p *pubSub) Close() error {
	return p.publisher.Close()
}
