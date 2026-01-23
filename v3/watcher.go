package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
)

// baseWatcher contains the common logic for both Watcher and WatcherEx.
type baseWatcher struct {
	pubsub       PubSub
	topic        string
	callbackFunc func(string)
	callbackMu   sync.RWMutex
	closed       chan struct{}
	logger       watermill.LoggerAdapter
}

// Option is a functional option for configuring the Watcher.
type Option func(*options)

type options struct {
	Topic  string
	Logger watermill.LoggerAdapter
}

// WithTopic sets the Watermill topic to use for updates.
func WithTopic(topic string) Option {
	return func(o *options) {
		o.Topic = topic
	}
}

// WithLogger sets the Watermill logger adapter to use.
func WithLogger(logger watermill.LoggerAdapter) Option {
	return func(o *options) {
		o.Logger = logger
	}
}

// newBaseWatcher creates a new base watcher instance.
func newBaseWatcher(ctx context.Context, connectionURL string, opts ...Option) (*baseWatcher, error) {
	o := &options{
		Topic:  "casbin-policy-updates",              // Default topic
		Logger: watermill.NewStdLogger(false, false), // Default logger
	}

	u, err := url.Parse(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// Priority 2: URL Path overrides default topic
	if u.Path != "" && u.Path != "/" {
		o.Topic = strings.TrimPrefix(u.Path, "/")
	}

	// Priority 3: Functional options override URL Path and default topic
	for _, opt := range opts {
		opt(o)
	}

	driversMu.RLock()
	driver, ok := drivers[u.Scheme]
	driversMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown driver scheme: %s", u.Scheme)
	}

	ps, err := driver.NewPubSub(ctx, u, o.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub for scheme %s: %w", u.Scheme, err)
	}

	w := &baseWatcher{
		pubsub: ps,
		topic:  o.Topic,
		closed: make(chan struct{}),
		logger: o.Logger,
	}

	if err := w.startSubscribe(ctx); err != nil {
		if closeErr := ps.Close(); closeErr != nil {
			w.logger.Error("failed to close pubsub", closeErr, nil)
		}
		return nil, err
	}

	return w, nil
}

func (w *baseWatcher) startSubscribe(ctx context.Context) error {
	messages, err := w.pubsub.Subscribe(ctx, w.topic)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					return
				}
				w.handleMessage(msg)
				msg.Ack()
			case <-w.closed:
				return
			}
		}
	}()

	return nil
}

func (w *baseWatcher) handleMessage(msg *message.Message) {
	w.callbackMu.RLock()
	callback := w.callbackFunc
	w.callbackMu.RUnlock()

	if callback == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panic in watcher callback", nil, watermill.LogFields{"panic": r})
		}
	}()

	callback(string(msg.Payload))
}

func (w *baseWatcher) SetUpdateCallback(callback func(string)) error {
	w.callbackMu.Lock()
	defer w.callbackMu.Unlock()
	w.callbackFunc = callback
	return nil
}

func (w *baseWatcher) Update() error {
	msg := message.NewMessage(watermill.NewUUID(), []byte("update"))
	return w.pubsub.Publish(w.topic, msg)
}

func (w *baseWatcher) Close() {
	close(w.closed)
	if err := w.pubsub.Close(); err != nil {
		w.logger.Error("failed to close pubsub", err, nil)
	}
}

// Watcher implements persist.Watcher.
type Watcher struct {
	*baseWatcher
}

// NewWatcher creates a new Watcher (basic mode).
func NewWatcher(ctx context.Context, connectionURL string, opts ...Option) (*Watcher, error) {
	base, err := newBaseWatcher(ctx, connectionURL, opts...)
	if err != nil {
		return nil, err
	}
	return &Watcher{baseWatcher: base}, nil
}

// WatcherEx implements persist.WatcherEx.
type WatcherEx struct {
	*baseWatcher
}

// NewWatcherEx creates a new WatcherEx (extended mode).
func NewWatcherEx(ctx context.Context, connectionURL string, opts ...Option) (*WatcherEx, error) {
	base, err := newBaseWatcher(ctx, connectionURL, opts...)
	if err != nil {
		return nil, err
	}
	return &WatcherEx{baseWatcher: base}, nil
}

// UpdateMessage represents the payload for WatcherEx updates.
type UpdateMessage struct {
	Type   string     `json:"type"`
	Sec    string     `json:"sec"`
	Ptype  string     `json:"ptype"`
	Params []string   `json:"params,omitempty"`
	Rules  [][]string `json:"rules,omitempty"`
}

func (w *WatcherEx) publishUpdate(u UpdateMessage) error {
	payload, err := json.Marshal(u)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewUUID(), payload)
	return w.pubsub.Publish(w.topic, msg)
}

func (w *WatcherEx) UpdateForAddPolicy(sec, ptype string, params ...string) error {
	return w.publishUpdate(UpdateMessage{Type: "add-policy", Sec: sec, Ptype: ptype, Params: params})
}

func (w *WatcherEx) UpdateForRemovePolicy(sec, ptype string, params ...string) error {
	return w.publishUpdate(UpdateMessage{Type: "remove-policy", Sec: sec, Ptype: ptype, Params: params})
}

func (w *WatcherEx) UpdateForRemoveFilteredPolicy(sec, ptype string, fieldIndex int, fieldValues ...string) error {
	return w.publishUpdate(UpdateMessage{
		Type:   "remove-filtered-policy",
		Sec:    sec,
		Ptype:  ptype,
		Params: append([]string{fmt.Sprintf("%d", fieldIndex)}, fieldValues...),
	})
}

// UpdateForSavePolicy is called when the policy is saved.
// This implementation sends a generic "save-policy" update notification.
// It does not send the full model to avoid large message payloads.
// The receiving Casbin enforcer is expected to reload the entire policy from the
// persistence layer (e.g., database) upon receiving this notification.
func (w *WatcherEx) UpdateForSavePolicy(_ model.Model) error {
	return w.publishUpdate(UpdateMessage{Type: "save-policy"})
}

func (w *WatcherEx) UpdateForAddPolicies(sec string, ptype string, rules ...[]string) error {
	return w.publishUpdate(UpdateMessage{Type: "add-policies", Sec: sec, Ptype: ptype, Rules: rules})
}

func (w *WatcherEx) UpdateForRemovePolicies(sec string, ptype string, rules ...[]string) error {
	return w.publishUpdate(UpdateMessage{Type: "remove-policies", Sec: sec, Ptype: ptype, Rules: rules})
}

// Ensure interfaces are implemented
var _ persist.Watcher = &Watcher{}
var _ persist.WatcherEx = &WatcherEx{}
