package watcher_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/casbin/casbin/v3"
	gnatsd "github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/origadmin/casbin-watcher/v3" // Corrected import path
	// Enable inmemory and NATS drivers
	_ "github.com/origadmin/casbin-watcher/v3/drivers/mem"  // Corrected import path
	_ "github.com/origadmin/casbin-watcher/v3/drivers/nats" // Corrected import path
)

func TestNATSWatcher(t *testing.T) {
	// Setup nats server
	s := gnatsd.RunDefaultServer()
	defer s.Shutdown()

	// Construct a valid URL for our NATS driver
	natsURL := fmt.Sprintf("nats://%s/casbin-topic?channel=casbin-channel", s.ClientURL())

	updaterCh := make(chan string, 1)
	listenerCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// updater represents the Casbin enforcer instance that changes the policy
	updater, err := watcher.NewWatcher(ctx, natsURL)
	require.NoError(t, err, "Failed to create updater")
	defer updater.Close()

	err = updater.SetUpdateCallback(func(msg string) {
		updaterCh <- "updater"
	})
	require.NoError(t, err)

	// listener represents any other Casbin enforcer instance that watches the change of policy
	listener, err := watcher.NewWatcher(ctx, natsURL)
	require.NoError(t, err, "Failed to create listener")
	defer listener.Close()

	// listener should set a callback that gets called when policy changes
	err = listener.SetUpdateCallback(func(msg string) {
		listenerCh <- "listener"
	})
	require.NoError(t, err)

	// updater changes the policy, and sends the notifications.
	err = updater.Update()
	require.NoError(t, err, "The updater failed to send Update")

	// Validate that listener received message
	// The updater should not receive its own message because it's a different connection.
	select {
	case res := <-listenerCh:
		require.Equal(t, "listener", res)
	case res := <-updaterCh:
		t.Fatalf("Updater should not receive its own message, but got: %s", res)
	case <-time.After(time.Second * 5):
		t.Fatal("Listener didn't receive message in time")
	}
}

func TestWithEnforcerNATS(t *testing.T) {
	// Setup nats server
	s := gnatsd.RunDefaultServer()
	defer s.Shutdown()

	natsURL := fmt.Sprintf("nats://%s/casbin-topic?channel=casbin-channel", s.ClientURL())

	updateCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := watcher.NewWatcher(ctx, natsURL)
	require.NoError(t, err, "Failed to create watcher")
	defer w.Close()

	// Initialize the enforcer.
	e, err := casbin.NewEnforcer("./test_data/model.conf", "./test_data/policy.csv")
	require.NoError(t, err)

	// Set the watcher for the enforcer.
	err = e.SetWatcher(w)
	require.NoError(t, err)

	// By default, the watcher's callback is automatically set to the
	// enforcer's LoadPolicy() in the SetWatcher() call.
	// We can change it by explicitly setting a callback.
	err = w.SetUpdateCallback(func(msg string) {
		updateCh <- "enforcer"
	})
	require.NoError(t, err)

	// Update the policy to test the effect.
	// This should call the watcher's Update method, which then calls our callback.
	err = e.SavePolicy()
	require.NoError(t, err)

	// Validate that listener received message
	select {
	case res := <-updateCh:
		require.Equal(t, "enforcer", res)
	case <-time.After(time.Second * 5):
		t.Fatal("The enforcer didn't send message in time")
	}
}

func TestWithEnforcerMemory(t *testing.T) {
	endpointURL := "mem://casbin?shared=true" // Use shared memory for this test

	updateCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := watcher.NewWatcher(ctx, endpointURL)
	require.NoError(t, err, "Failed to create watcher")
	defer w.Close()

	e, err := casbin.NewEnforcer("../test_data/model.conf", "../test_data/policy.csv")
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

func TestWithEnforcerMemory_NonShared(t *testing.T) {
	// Two watchers with the same topic but non-shared memory should not communicate.
	endpointURL := "mem://casbin?shared=false"

	updaterCh := make(chan string, 1)
	listenerCh := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Updater
	updaterWatcher, err := watcher.NewWatcher(ctx, endpointURL)
	require.NoError(t, err)
	defer updaterWatcher.Close()
	err = updaterWatcher.SetUpdateCallback(func(msg string) {
		updaterCh <- "updater"
	})
	require.NoError(t, err)

	// Listener
	listenerWatcher, err := watcher.NewWatcher(ctx, endpointURL)
	require.NoError(t, err)
	defer listenerWatcher.Close()
	err = listenerWatcher.SetUpdateCallback(func(msg string) {
		listenerCh <- "listener"
	})
	require.NoError(t, err)

	// Updater sends an update
	err = updaterWatcher.Update()
	require.NoError(t, err)

	// Validate that the listener does NOT receive the message.
	select {
	case res := <-listenerCh:
		t.Fatalf("Listener should not have received a message, but got: %s", res)
	case res := <-updaterCh:
		t.Fatalf("Updater should not receive its own message, but got: %s", res)
	case <-time.After(time.Millisecond * 100):
		// Test passed, no message received in a short time.
	}
}
