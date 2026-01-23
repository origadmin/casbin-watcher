# HTTP Driver for Casbin Watcher

This directory contains the HTTP (`http`) driver for `casbin-watcher`. This driver uses standard HTTP POST requests to
publish policy updates.

## How it Works

The `http` driver is built on Watermill's `http` Pub/Sub implementation.

- **Publisher-Only**: This driver **only implements the publisher side**. It sends policy updates as HTTP POST requests
  to a configurable base URL.
- **Subscriber Implementation**: The subscriber side is **not implemented** within this driver. Receiving messages via
  HTTP requires setting up a dedicated HTTP server and integrating it with Watermill's `http.Router`. This is beyond the
  scope of a simple driver that is instantiated via a URL.
- **Use Case**: This driver is useful for sending policy updates to a web service that is already equipped to handle
  incoming webhooks.

## Configuration

The driver is configured using a URL.

### URL Format

```
http://?base_url=http://your-service.com/webhooks
```

- **Scheme**: The scheme must be `http`.
- **Parameters**: The base URL for the webhook endpoint is configured via a query parameter.

### Configuration Parameters

| Parameter  | Type     | Default | Description                                                                                          | Example                                  |
|------------|----------|---------|------------------------------------------------------------------------------------------------------|------------------------------------------|
| `base_url` | `string` | (none)  | **Required.** The base URL to which messages will be POSTed. The topic will be appended to this URL. | `base_url=http://api.example.com/casbin` |

### Usage Example

```go
import (
    "context"
    "log"

    "github.com/casbin/casbin/v2"
    "github.com/origadmin/casbin-watcher/v3"
    _ "github.com/origadmin/casbin-watcher/v3/drivers/http" // Register the driver
)

func main() {
    // The watcher will POST messages to "http://localhost:8080/casbin_updates".
    connectionURL := "http://?base_url=http://localhost:8080"
    
    w, err := watcher.NewWatcher(context.Background(), connectionURL, "casbin_updates")
    if err != nil {
        log.Fatalf("Failed to create watcher: %v", err)
    }

    e, err := casbin.NewEnforcer("model.conf", "policy.csv")
    if err != nil {
        log.Fatalf("Failed to create enforcer: %v", err)
    }

    err = e.SetWatcher(w)
    if err != nil {
        log.Fatalf("Failed to set watcher: %v", err)
    }
    
    // When you call e.SavePolicy(), a POST request will be sent.
}
```

### Implementing an HTTP Subscriber

To receive messages published by this driver, you need to set up an HTTP server and integrate it with Watermill's
`http.Router`. Here's a simplified example of how you might do this (refer to Watermill's documentation for full
details):

```go
package main

import (
	"context"
	"log"
	stdHttp "net/http"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-http/v2/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
)

func main() {
	logger := watermill.NewStdLogger(true, true)
	ctx := context.Background()

	// Create a dummy publisher to satisfy the router's handler signature.
	// In a real scenario, you might forward messages to another Pub/Sub.
	dummyPublisher, err := message.NewPublisher(message.PublisherConfig{}, logger)
	if err != nil {
		log.Fatal(err)
	}

	httpSubscriber, err := http.NewSubscriber(
		":8080", // The address your server will listen on
		http.SubscriberConfig{
			UnmarshalMessageFunc: func(topic string, request *stdHttp.Request) (*message.Message, error) {
				// Implement your message unmarshalling logic here.
				// For example, read the request body as the message payload.
				payload, err := io.ReadAll(request.Body)
				if err != nil {
					return nil, err
				}
				return message.NewMessage(watermill.NewUUID(), payload), nil
			},
		},
		logger,
	)
	if err != nil {
		log.Fatal(err)
	}

	r, err := router.NewRouter(router.Config{}, logger)
	if err != nil {
		log.Fatal(err)
	}

	r.AddMiddleware(
		middleware.Recoverer,
		middleware.CorrelationID,
	)
	r.AddPlugin(plugin.SignalsHandler)

	// Define a handler for incoming messages on the "/casbin_updates" path.
	r.AddHandler(
		"casbin_webhook_handler",
		"/casbin_updates", // This is the URL path that the publisher will POST to
		httpSubscriber,
		"output_topic", // This is a dummy topic for the router's internal use
		dummyPublisher,
		func(msg *message.Message) ([]*message.Message, error) {
			log.Printf("Received message from HTTP: %s, Payload: %s", msg.UUID, string(msg.Payload))
			// Process the message here.
			// Acknowledge the message.
			msg.Ack()
			return nil, nil // No messages to publish further
		},
	)

	// Start the HTTP server in a goroutine.
	go func() {
		<-r.Running() // Wait for the router to start
		log.Printf("HTTP server listening on %s", httpSubscriber.Addr())
		if err := httpSubscriber.StartHTTPServer(); err != nil && err != stdHttp.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	log.Println("Starting Watermill router...")
	if err := r.Run(ctx); err != nil {
		log.Fatalf("Watermill router failed: %v", err)
	}
}
```
