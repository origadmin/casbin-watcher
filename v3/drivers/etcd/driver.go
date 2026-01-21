package etcd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.etcd.io/etcd/client/v3"

	"github.com/origadmin/casbin-watcher/v3"
)

func init() {
	watcher.RegisterDriver("etcd", &Driver{})
}

type Driver struct{}

func (d *Driver) NewPubSub(ctx context.Context, u *url.URL, logger watermill.LoggerAdapter) (watcher.PubSub, error) {
	config, err := parseEtcdURL(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd url: %w", err)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	publisher := &EtcdPublisher{cli: cli, logger: logger}
	subscriber := &EtcdSubscriber{cli: cli, logger: logger}

	return &etcdPubSub{publisher: publisher, subscriber: subscriber}, nil
}

type etcdPubSub struct {
	publisher  *EtcdPublisher
	subscriber *EtcdSubscriber
}

func (e *etcdPubSub) Publish(topic string, messages ...*message.Message) error {
	return e.publisher.Publish(topic, messages...)
}

func (e *etcdPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return e.subscriber.Subscribe(ctx, topic)
}

func (e *etcdPubSub) Close() error {
	errPub := e.publisher.Close()
	errSub := e.subscriber.Close()
	if errPub != nil {
		return errPub
	}
	return errSub
}

// EtcdPublisher implements message.Publisher for etcd.
type EtcdPublisher struct {
	cli    *clientv3.Client
	logger watermill.LoggerAdapter
}

func (p *EtcdPublisher) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		key := topic + "/" + msg.UUID
		_, err := p.cli.Put(context.Background(), key, string(msg.Payload))
		if err != nil {
			p.logger.Error("Failed to publish message to etcd", err, watermill.LogFields{"topic": topic, "key": key})
			return err
		}
	}
	return nil
}

func (p *EtcdPublisher) Close() error {
	return p.cli.Close()
}

// EtcdSubscriber implements message.Subscriber for etcd.
type EtcdSubscriber struct {
	cli    *clientv3.Client
	logger watermill.LoggerAdapter
}

func (s *EtcdSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	output := make(chan *message.Message)
	watchChan := s.cli.Watch(ctx, topic, clientv3.WithPrefix())

	go func() {
		defer close(output)
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("Etcd subscriber context done", watermill.LogFields{"topic": topic})
				return
			case watchResp, ok := <-watchChan:
				if !ok {
					s.logger.Error("Etcd watch channel closed", nil, watermill.LogFields{"topic": topic})
					return
				}
				if watchResp.Err() != nil {
					s.logger.Error("Etcd watch error", watchResp.Err(), watermill.LogFields{"topic": topic})
					continue
				}
				for _, event := range watchResp.Events {
					if event.Type == clientv3.EventTypePut {
						msg := message.NewMessage(watermill.NewUUID(), event.Kv.Value)
						output <- msg
					}
				}
			}
		}
	}()

	return output, nil
}

func (s *EtcdSubscriber) Close() error {
	return s.cli.Close()
}

type etcdConfig struct {
	Endpoints []string
}

func parseEtcdURL(u *url.URL) (*etcdConfig, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("etcd endpoints are not specified in URL host")
	}

	config := &etcdConfig{
		Endpoints: strings.Split(u.Host, ","),
	}

	return config, nil
}
