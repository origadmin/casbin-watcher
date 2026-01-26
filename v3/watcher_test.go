package watcher_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/casbin/casbin/v3"
	gnatsd "github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/origadmin/casbin-watcher/v3"
	_ "github.com/origadmin/casbin-watcher/v3/drivers/mem"
	_ "github.com/origadmin/casbin-watcher/v3/drivers/nats"
)

func TestNATSWatcher(t *testing.T) {
	s := gnatsd.RunDefaultServer()
	defer s.Shutdown()

	parsedURL, err := url.Parse(s.ClientURL())
	require.NoError(t, err)
	natsURL := fmt.Sprintf("nats://%s/casbin-topic?channel=casbin-channel", parsedURL.Host)

	listenerCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updater, err := watcher.NewWatcher(ctx, natsURL)
	require.NoError(t, err, "Failed to create updater")
	defer updater.Close()

	listener, err := watcher.NewWatcher(ctx, natsURL)
	require.NoError(t, err, "Failed to create listener")
	defer listener.Close()

	err = listener.SetUpdateCallback(func(msg string) {
		listenerCh <- "listener"
	})
	require.NoError(t, err)

	err = updater.Update()
	require.NoError(t, err, "The updater failed to send Update")

	select {
	case res := <-listenerCh:
		require.Equal(t, "listener", res)
	case <-time.After(time.Second * 5):
		t.Fatal("Listener didn't receive message in time")
	}
}

func testWithEnforcer(t *testing.T, endpointURL string) {
	updateCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := watcher.NewWatcher(ctx, endpointURL)
	require.NoError(t, err, "Failed to create watcher")
	defer w.Close()

	e, err := casbin.NewEnforcer("./test_data/model.conf", "./test_data/policy.csv")
	require.NoError(t, err)

	err = e.SetWatcher(w)
	require.NoError(t, err)

	err = w.SetUpdateCallback(func(msg string) {
		updateCh <- "enforcer"
	})
	require.NoError(t, err)

	err = e.SavePolicy()
	require.NoError(t, err)

	select {
	case res := <-updateCh:
		require.Equal(t, "enforcer", res)
	case <-time.After(time.Second * 5):
		t.Fatal("The enforcer didn't send message in time")
	}
}

func TestWithEnforcerNATS(t *testing.T) {
	s := gnatsd.RunDefaultServer()
	defer s.Shutdown()

	parsedURL, err := url.Parse(s.ClientURL())
	require.NoError(t, err)
	natsURL := fmt.Sprintf("nats://%s/casbin-topic?channel=casbin-channel", parsedURL.Host)

	testWithEnforcer(t, natsURL)
}

func TestWithEnforcerMemory(t *testing.T) {
	endpointURL := "mem://casbin?shared=true"
	testWithEnforcer(t, endpointURL)
}

func TestWithEnforcerMemory_NonShared(t *testing.T) {
	endpointURL := "mem://casbin?shared=false"

	listenerCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updaterWatcher, err := watcher.NewWatcher(ctx, endpointURL)
	require.NoError(t, err)
	defer updaterWatcher.Close()

	listenerWatcher, err := watcher.NewWatcher(ctx, endpointURL)
	require.NoError(t, err)
	defer listenerWatcher.Close()
	err = listenerWatcher.SetUpdateCallback(func(msg string) {
		listenerCh <- "listener"
	})
	require.NoError(t, err)

	err = updaterWatcher.Update()
	require.NoError(t, err)

	select {
	case res := <-listenerCh:
		t.Fatalf("Listener should NOT have received a message in non-shared mode, but got: %s", res)
	case <-time.After(time.Millisecond * 200):
		// Test passed.
	}
}

func TestWatcherEx(t *testing.T) {
	endpointURL := "mem://casbin?shared=true"

	tests := []struct {
		name  string
		codec watcher.MarshalUnmarshaler
	}{
		{
			name:  "GOB Codec",
			codec: watcher.DefaultCodec(),
		},
		{
			name:  "JSON Codec",
			codec: watcher.JSONCodec(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			updateCh := make(chan watcher.UpdateMessage, 1)

			updater, err := watcher.NewWatcherEx(ctx, endpointURL, watcher.WithCodec(tt.codec))
			require.NoError(t, err)
			defer updater.Close()

			listener, err := watcher.NewWatcher(ctx, endpointURL)
			require.NoError(t, err)
			defer listener.Close()

			err = listener.SetUpdateCallback(func(msg string) {
				var updateMsg watcher.UpdateMessage
				err := tt.codec.Unmarshal([]byte(msg), &updateMsg)
				require.NoError(t, err)
				updateCh <- updateMsg
			})
			require.NoError(t, err)

			// Test UpdateForAddPolicy
			err = updater.UpdateForAddPolicy("p", "p", "alice", "data1", "read")
			require.NoError(t, err)

			select {
			case updateMsg := <-updateCh:
				require.Equal(t, watcher.UpdateTypeAddPolicy, updateMsg.Type)
				require.Equal(t, "p", updateMsg.Sec)
				require.Equal(t, "p", updateMsg.Ptype)
				require.Equal(t, []string{"alice", "data1", "read"}, updateMsg.Params)
			case <-time.After(time.Second * 5):
				t.Fatal("Listener didn't receive message for AddPolicy in time")
			}

			// Test UpdateForRemovePolicy
			err = updater.UpdateForRemovePolicy("p", "p", "bob", "data2", "write")
			require.NoError(t, err)

			select {
			case updateMsg := <-updateCh:
				require.Equal(t, watcher.UpdateTypeRemovePolicy, updateMsg.Type)
				require.Equal(t, []string{"bob", "data2", "write"}, updateMsg.Params)
			case <-time.After(time.Second * 5):
				t.Fatal("Listener didn't receive message for RemovePolicy in time")
			}
		})
	}
}
