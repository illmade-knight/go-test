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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	projectID := "test-project-pubsub"
	runID := uuid.NewString()
	topicID := fmt.Sprintf("test-topic-%s", runID)
	subID := fmt.Sprintf("test-subscription-%s", runID)
	cfg := emulators.GetDefaultPubsubConfig(projectID)

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

	subscriber := client.Subscriber(subID)
	go func() {
		err := subscriber.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			received <- msg.Data
			msg.Ack()
		})

		// --- START: GOROUTINE ERROR FIX ---
		// This goroutine will always receive an error when the test
		// context is canceled. We must check for both standard
		// context.Canceled and gRPC's Canceled status code.
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return // This is an expected error
			}
			if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
				return // This is also an expected error
			}
			// Any other error is a real failure.
			t.Errorf("Receive error: %v", err)
		}
		// --- END: GOROUTINE ERROR FIX ---
	}()

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

// ... TestSetupFirestoreEmulator remains the same ...
func TestSetupFirestoreEmulator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	projectID := "test-project-firestore"
	cfg := emulators.GetDefaultFirestoreConfig(projectID)

	connInfo := emulators.SetupFirestoreEmulator(t, context.Background(), cfg)

	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint should not be empty")
	require.NotEmpty(t, connInfo.ClientOptions, "ClientOptions should not be empty")

	client, err := firestore.NewClient(ctx, projectID, connInfo.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})

	_, _, err = client.Collection("testCollection").Add(ctx, map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	})
	require.NoError(t, err, "Failed to add document to Firestore")

	t.Logf("Firestore emulator test passed. Connected to: %s", connInfo.HTTPEndpoint.Endpoint)
}
