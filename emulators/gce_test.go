package emulators_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/google/uuid"
	"github.com/illmade-knight/go-test/emulators"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// createPubsubResources helper remains the same.
func createPubsubResources(t *testing.T, ctx context.Context, client *pubsub.Client, projectID, topicID, subID string) {
	t.Helper()
	topicAdmin := client.TopicAdminClient
	subAdmin := client.SubscriptionAdminClient
	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	_, err := topicAdmin.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = topicAdmin.DeleteTopic(context.Background(), &pubsubpb.DeleteTopicRequest{Topic: topicName})
	})
	subName := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subID)
	_, err = subAdmin.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:  subName,
		Topic: topicName,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = subAdmin.DeleteSubscription(context.Background(), &pubsubpb.DeleteSubscriptionRequest{Subscription: subName})
	})
}

func TestSetupPubsubEmulator(t *testing.T) {
	t.Parallel()

	// This is the overall test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	projectID := "test-project-pubsub"
	runID := uuid.NewString()
	topicID := fmt.Sprintf("test-topic-%s", runID)
	subID := fmt.Sprintf("test-subscription-%s", runID)
	cfg := emulators.GetDefaultPubsubConfig(projectID)

	// Use background context for setup
	connInfo := emulators.SetupPubsubEmulator(t, context.Background(), cfg)

	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint should not be empty")
	require.NotEmpty(t, connInfo.ClientOptions, "ClientOptions should not be empty")

	client, err := pubsub.NewClient(ctx, projectID, connInfo.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})

	createPubsubResources(t, ctx, client, projectID, topicID, subID)

	// Polling loop remains the same.
	t.Logf("Polling for subscription %s to exist...", subID)
	subAdmin := client.SubscriptionAdminClient
	subName := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subID)
	req := &pubsubpb.GetSubscriptionRequest{Subscription: subName}
	pollCtx, pollCancel := context.WithTimeout(ctx, 30*time.Second)
	defer pollCancel()

	for {
		_, err := subAdmin.GetSubscription(pollCtx, req)
		if err == nil {
			t.Logf("Subscription %s confirmed to exist.", subID)
			break
		}
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			select {
			case <-time.After(200 * time.Millisecond):
				continue
			case <-pollCtx.Done():
				t.Fatalf("Timed out waiting for subscription %s to exist. Last error: %v", subID, err)
			}
		}
		t.Fatalf("Error while polling for subscription %s: %v", subID, err)
	}

	received := make(chan []byte, 1)

	// --- START: GOROUTINE FIX ---
	// Create a new cancellable context *just for the receiver*.
	receiveCtx, cancelReceive := context.WithCancel(ctx)
	// Stop the receiver goroutine *before* other cleanups (like deleting the sub) run.
	t.Cleanup(cancelReceive)

	subscriber := client.Subscriber(subID)
	go func() {
		// Use the new receiveCtx here
		err := subscriber.Receive(receiveCtx, func(ctx context.Context, msg *pubsub.Message) {
			received <- msg.Data
			msg.Ack()
			cancelReceive() // Stop the receiver *immediately* after getting msg
		})

		// This error check now correctly ignores context canceled errors.
		if err != nil && !errors.Is(err, context.Canceled) {
			if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
				return // This is expected
			}
		}
	}()
	// --- END: GOROUTINE FIX ---

	publisher := client.Publisher(topicID)
	defer publisher.Stop()
	res := publisher.Publish(ctx, &pubsub.Message{Data: []byte("hello world")})
	_, err = res.Get(ctx)
	require.NoError(t, err, "Failed to publish message")

	select {
	case msg := <-received:
		require.Equal(t, "hello world", string(msg))
	case <-ctx.Done():
		t.Fatalf("Test timed out waiting for message: %v", ctx.Err())
	}

	t.Logf("Pub/Sub emulator test passed. Connected to: %s", connInfo.HTTPEndpoint.Endpoint)
}

func TestSetupFirestoreEmulator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	projectID := "test-project-firestore"
	cfg := emulators.GetDefaultFirestoreConfig(projectID)

	// This call will now block until the Firestore emulator is *actually*
	// ready, thanks to the fix in gce.go.
	connInfo := emulators.SetupFirestoreEmulator(t, context.Background(), cfg)

	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint should not be empty")
	require.NotEmpty(t, connInfo.ClientOptions, "ClientOptions should not be empty")

	client, err := firestore.NewClient(ctx, projectID, connInfo.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})

	// This write operation will no longer fail with DeadlineExceeded,
	// because SetupFirestoreEmulator already guaranteed the service was ready.
	_, _, err = client.Collection("testCollection").Add(ctx, map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	})
	require.NoError(t, err, "Failed to add document to Firestore")

	t.Logf("Firestore emulator test passed. Connected to: %s", connInfo.HTTPEndpoint.Endpoint)
}

// --- NEW DUAL EMULATOR TEST ---

// TestSetupDualEmulators verifies that both emulators can be started and
// used concurrently within the same test.
func TestSetupDualEmulators(t *testing.T) {
	// t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	projectID := "test-project-dual"

	// 1. Setup both emulators
	// Use background context for setup. These functions now block until ready.
	fsCfg := emulators.GetDefaultFirestoreConfig(projectID)
	fsConnInfo := emulators.SetupFirestoreEmulator(t, context.Background(), fsCfg)

	psCfg := emulators.GetDefaultPubsubConfig(projectID)
	psConnInfo := emulators.SetupPubsubEmulator(t, context.Background(), psCfg)

	// 2. Create both clients
	fsClient, err := firestore.NewClient(ctx, projectID, fsConnInfo.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = fsClient.Close()
	})

	psClient, err := pubsub.NewClient(ctx, projectID, psConnInfo.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = psClient.Close()
	})

	// 3. Perform a write operation on Firestore
	t.Log("Performing Firestore write...")
	_, _, err = fsClient.Collection("dualTest").Add(ctx, map[string]interface{}{
		"service": "firestore",
	})
	require.NoError(t, err, "Failed to write to Firestore in dual test")
	t.Log("Firestore write successful.")

	// 4. Perform a write operation on Pub/Sub
	t.Log("Performing Pub/Sub write...")
	topicID := "dual-test-topic"
	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	_, err = psClient.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	require.NoError(t, err, "Failed to create topic in dual test")
	t.Log("Pub/Sub write successful.")

	t.Log("✅ Dual emulator test passed.")
}
