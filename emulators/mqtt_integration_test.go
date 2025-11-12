package emulators

import (
	"context"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

// TestMqttPublishSubscribeIntegration tests the full publish-subscribe flow
// using the Mosquitto emulator and MQTT clients.
func TestMqttPublishSubscribeIntegration(t *testing.T) {
	t.Parallel() // Allow this test to run in parallel with others

	// 1. Setup Mosquitto Emulator
	t.Log("Setting up Mosquitto emulator...")
	cfg := GetDefaultMqttImageContainer()
	connInfo := SetupMosquittoContainer(t, context.Background(), cfg)
	brokerURL := connInfo.EmulatorAddress
	require.NotEmpty(t, brokerURL, "EmulatorAddress should not be empty")
	t.Logf("Mosquitto emulator running at: %s", brokerURL)

	testTopic := "test/messages"
	testMessage := "Hello MQTT from Testcontainers!"
	receivedMessage := make(chan string, 1) // Channel to receive the message

	var wg sync.WaitGroup
	wg.Add(1) // Wait for the subscriber to receive a message

	// 2. Create and connect MQTT Subscriber
	subscriberClientID := "test-subscriber-client"
	subOpts := mqtt.NewClientOptions().AddBroker(brokerURL).SetClientID(subscriberClientID).SetAutoReconnect(false)
	subscriber := mqtt.NewClient(subOpts)

	t.Log("Connecting MQTT subscriber...")
	connectToken := subscriber.Connect()
	// Wait for connection or timeout
	if !connectToken.WaitTimeout(10 * time.Second) {
		t.Fatal("MQTT subscriber connection timed out")
	}
	require.NoError(t, connectToken.Error(), "Failed to connect MQTT subscriber")

	t.Cleanup(func() {
		subscriber.Disconnect(250)
	})
	require.True(t, subscriber.IsConnected(), "Subscriber should be connected")

	t.Logf("Subscriber connected. Subscribing to topic: %s", testTopic)
	subscribeToken := subscriber.Subscribe(testTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		t.Logf("Subscriber received message: %s on topic: %s", msg.Payload(), msg.Topic())
		// Send to channel in a non-blocking way in case test already timed out
		select {
		case receivedMessage <- string(msg.Payload()):
		default:
			t.Log("Message received, but channel was full or closed.")
		}
		wg.Done() // Signal that the message was received
	})
	if !subscribeToken.WaitTimeout(5 * time.Second) {
		t.Fatal("MQTT subscriber subscribe timed out")
	}
	require.NoError(t, subscribeToken.Error(), "Failed to subscribe")

	// 3. Create and connect MQTT Publisher
	publisherClientID := "test-publisher-client"
	publisher, err := CreateTestMqttPublisher(brokerURL, publisherClientID)
	require.NoError(t, err, "Failed to create MQTT publisher")
	t.Cleanup(func() {
		publisher.Disconnect(250)
	})

	require.True(t, publisher.IsConnected(), "Publisher should be connected")
	t.Log("Publisher connected.")

	// 4. Publish a message
	t.Logf("Publisher sending message: %s to topic: %s", testMessage, testTopic)
	publishToken := publisher.Publish(testTopic, 0, false, testMessage)
	if !publishToken.WaitTimeout(5 * time.Second) {
		t.Fatal("MQTT publish timed out")
	}
	require.NoError(t, publishToken.Error(), "Failed to publish message")
	t.Log("Message published.")

	// 5. Wait for the message to be received by the subscriber
	select {
	case msg := <-receivedMessage:
		require.Equal(t, testMessage, msg, "Received message content mismatch")
	case <-time.After(15 * time.Second): // Give some time for message propagation
		t.Fatal("Timed out waiting for message to be received by subscriber")
	}

	wg.Wait() // Ensure the handler has completed

	t.Log("MQTT publish/subscribe integration test completed successfully.")
}
