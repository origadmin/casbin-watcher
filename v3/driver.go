package watcher

import (
	"context"
	"net/url"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
)

// Driver is the interface for creating a PubSub instance for a specific backend.
// Each driver is responsible for parsing its specific configuration from the URL.
type Driver interface {
	// NewPubSub creates a new PubSub instance from the parsed URL and logger.
	// The driver is responsible for parsing and validating its specific configuration
	// from the URL's host, path, and query parameters.
	NewPubSub(ctx context.Context, parsedURL *url.URL, logger watermill.LoggerAdapter) (PubSub, error)
}

// Global driver registry
var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

// RegisterDriver registers a driver for the given scheme (e.g., "redis", "kafka").
// This allows casbin-watcher to dynamically find and create PubSub instances.
func RegisterDriver(scheme string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("watcher: Register driver is nil")
	}
	if _, dup := drivers[scheme]; dup {
		panic("watcher: Register called twice for driver " + scheme)
	}
	drivers[scheme] = driver
}
