package watcher

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
)

// DefaultTopic is the default topic used for policy update notifications.
const DefaultTopic = "casbin-policy-updates"

// Update types for Ex messages.
const (
	UpdateTypePolicyChanged        = "policy-changed"
	UpdateTypeAddPolicy            = "add-policy"
	UpdateTypeRemovePolicy         = "remove-policy"
	UpdateTypeRemoveFilteredPolicy = "remove-filtered-policy"
	UpdateTypeSavePolicy           = "save-policy"
	UpdateTypeAddPolicies          = "add-policies"
	UpdateTypeRemovePolicies       = "remove-policies"
)

// baseWatcher contains the common logic for both Watcher and Ex.
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
	Codec  MarshalUnmarshaler // Codec is only used by Ex, but passed via options
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

// WithCodec sets the custom MarshalUnmarshaler for UpdateMessage.
func WithCodec(codec MarshalUnmarshaler) Option {
	return func(o *options) {
		o.Codec = codec
	}
}

// newBaseWatcher creates a new base watcher instance with the provided options.
func newBaseWatcher(ctx context.Context, connectionURL string, o *options) (*baseWatcher, error) {
	u, err := url.Parse(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
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

// parseOptions parses and merges options from URL and functional options.
func parseOptions(connectionURL string, opts ...Option) (*options, error) {
	o := &options{
		Topic:  "",                                   // Empty by default, will be set from URL or options
		Logger: watermill.NewStdLogger(false, false), // Default logger
		Codec:  &gobMarshalUnmarshaler{},             // Default codec (only used by Ex)
	}

	u, err := url.Parse(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// Priority: WithTopic > URL Path > Default
	// First, apply URL path if it exists
	if u.Path != "" && u.Path != "/" {
		o.Topic = strings.TrimPrefix(u.Path, "/")
	}
	// Then, functional options can override URL path
	for _, opt := range opts {
		opt(o)
	}

	// If still not set, use default
	if o.Topic == "" {
		o.Topic = DefaultTopic
	}

	return o, nil
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

	// For Watcher (basic mode), the payload is expected to be a simple string.
	// For WatcherEx, the payload is an UpdateMessage, but the callbackFunc is for Watcher.
	// The WatcherEx does not use callbackFunc for receiving updates.
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

// closeInternal performs the actual close operation and returns any error.
// This is an internal helper to allow proper error handling without changing the public interface.
func (w *baseWatcher) closeInternal() error {
	select {
	case <-w.closed:
		// Already closed
		return nil
	default:
	}

	close(w.closed)
	if err := w.pubsub.Close(); err != nil {
		w.logger.Error("failed to close pubsub", err, nil)
		return err
	}
	return nil
}

// Close stops and releases the watcher.
// Public interface method - errors are logged but not returned to match persist.Watcher interface.
func (w *baseWatcher) Close() {
	_ = w.closeInternal()
}

// Watcher implements persist.Watcher.
type Watcher struct {
	*baseWatcher
}

// NewWatcher creates a new Watcher (basic mode).
func NewWatcher(ctx context.Context, connectionURL string, opts ...Option) (*Watcher, error) {
	o, err := parseOptions(connectionURL, opts...)
	if err != nil {
		return nil, err
	}
	base, err := newBaseWatcher(ctx, connectionURL, o)
	if err != nil {
		return nil, err
	}
	return &Watcher{baseWatcher: base}, nil
}

// Ex implements persist.WatcherEx.
type Ex struct {
	*baseWatcher
	codec MarshalUnmarshaler // Codec is specific to Ex
}

// NewWatcherEx creates a new Ex (extended mode).
func NewWatcherEx(ctx context.Context, connectionURL string, opts ...Option) (*Ex, error) {
	o, err := parseOptions(connectionURL, opts...)
	if err != nil {
		return nil, err
	}
	base, err := newBaseWatcher(ctx, connectionURL, o)
	if err != nil {
		return nil, err
	}
	return &Ex{
		baseWatcher: base,
		codec:       o.Codec,
	}, nil
}

// UpdateMessage represents the payload for Ex updates.
type UpdateMessage struct {
	Type   string
	Sec    string
	Ptype  string
	Params []string
	Rules  [][]string
}

func (w *Ex) publishUpdate(u UpdateMessage) error {
	payload, err := w.codec.Marshal(u)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewUUID(), payload)
	return w.pubsub.Publish(w.topic, msg)
}

// Update calls the update callback of other instances to synchronize their policy.
// This method is part of the Watcher interface and publishes a generic "policy-changed" message.
func (w *Ex) Update() error {
	return w.publishUpdate(UpdateMessage{Type: UpdateTypePolicyChanged})
}

func (w *Ex) UpdateForAddPolicy(sec, ptype string, params ...string) error {
	return w.publishUpdate(UpdateMessage{Type: UpdateTypeAddPolicy, Sec: sec, Ptype: ptype, Params: params})
}

func (w *Ex) UpdateForRemovePolicy(sec, ptype string, params ...string) error {
	return w.publishUpdate(UpdateMessage{Type: UpdateTypeRemovePolicy, Sec: sec, Ptype: ptype, Params: params})
}

func (w *Ex) UpdateForRemoveFilteredPolicy(sec, ptype string, fieldIndex int, fieldValues ...string) error {
	return w.publishUpdate(UpdateMessage{
		Type:   UpdateTypeRemoveFilteredPolicy,
		Sec:    sec,
		Ptype:  ptype,
		Params: append([]string{fmt.Sprintf("%d", fieldIndex)}, fieldValues...),
	})
}

func (w *Ex) UpdateForSavePolicy(_ model.Model) error {
	return w.publishUpdate(UpdateMessage{Type: UpdateTypeSavePolicy})
}

func (w *Ex) UpdateForAddPolicies(sec string, ptype string, rules ...[]string) error {
	return w.publishUpdate(UpdateMessage{Type: UpdateTypeAddPolicies, Sec: sec, Ptype: ptype, Rules: rules})
}

func (w *Ex) UpdateForRemovePolicies(sec string, ptype string, rules ...[]string) error {
	return w.publishUpdate(UpdateMessage{Type: UpdateTypeRemovePolicies, Sec: sec, Ptype: ptype, Rules: rules})
}

// Ensure interfaces are implemented
var _ persist.Watcher = &Watcher{}
var _ persist.WatcherEx = &Ex{}
