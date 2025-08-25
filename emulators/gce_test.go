package emulators_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	// REFACTOR: Use the v2 pubsub import path.
	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/google/uuid"
	"github.com/illmade-knight/go-test/emulators"
	"github.com/stretchr/testify/require"
)

// REFACTOR: createPubsubResources now correctly uses the admin clients from the
// main pubsub.Client, as shown in the v2 documentation.
func createPubsubResources(t *testing.T, ctx context.Context, client *pubsub.Client, projectID, topicID, subID string) {
	t.Helper()

	// REFACTOR: Access the admin clients directly from the operational client.
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

	connInfo := emulators.SetupPubsubEmulator(t, ctx, cfg)

	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint should not be empty")
	require.NotEmpty(t, connInfo.HTTPEndpoint.Port, "HTTPEndpoint.Port should not be empty")
	require.NotEmpty(t, connInfo.ClientOptions, "ClientOptions should not be empty")

	client, err := pubsub.NewClient(ctx, projectID, connInfo.ClientOptions...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})

	// REFACTOR: Call the dedicated helper to create the test resources.
	createPubsubResources(t, ctx, client, projectID, topicID, subID)

	var wg sync.WaitGroup
	wg.Add(1)
	received := make(chan []byte, 1)

	subscriber := client.Subscriber(subID)
	go func() {
		defer wg.Done()
		err := subscriber.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			received <- msg.Data
			msg.Ack()
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Receive error: %v", err)
		}
	}()

	publisher := client.Publisher(topicID)
	defer publisher.Stop()
	res := publisher.Publish(ctx, &pubsub.Message{Data: []byte("hello world")})
	_, err = res.Get(ctx)
	require.NoError(t, err)

	select {
	case msg := <-received:
		require.Equal(t, "hello world", string(msg))
	case <-ctx.Done():
		t.Fatal("Test timed out waiting for message")
	}

	wg.Wait()
	t.Logf("Pub/Sub emulator test passed. Connected to: %s", connInfo.HTTPEndpoint.Endpoint)
}

func TestSetupFirestoreEmulator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	projectID := "test-project-firestore"
	cfg := emulators.GetDefaultFirestoreConfig(projectID)

	connInfo := emulators.SetupFirestoreEmulator(t, ctx, cfg)

	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint should not be empty")
	require.NotEmpty(t, connInfo.HTTPEndpoint.Port, "HTTPEndpoint.Port should not be empty")
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
