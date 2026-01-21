package watcher

import "github.com/ThreeDotsLabs/watermill/message"

// PubSub is a high-level interface that combines Publisher and Subscriber.
// This is the standard interface that all drivers must produce.
type PubSub interface {
	message.Publisher
	message.Subscriber
}
