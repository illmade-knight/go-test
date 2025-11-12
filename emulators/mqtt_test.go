package emulators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require" // Using require for fatal assertions
)

func TestSetupMosquittoContainer(t *testing.T) {
	t.Parallel() // Allow tests to run in parallel

	cfg := GetDefaultMqttImageContainer()
	// Pass context.Background() to SetupMosquittoContainer for container lifecycle
	connInfo := SetupMosquittoContainer(t, context.Background(), cfg)

	// --- Verify EmulatorConnectionInfo ---
	require.NotEmpty(t, connInfo.EmulatorAddress, "EmulatorAddress is empty")

	// --- Test Connectivity ---
	clientID := "test-publisher-client"
	// CreateTestMqttPublisher handles its own timeouts internally with token.WaitTimeout
	publisher, err := CreateTestMqttPublisher(connInfo.EmulatorAddress, clientID)
	require.NoError(t, err, "Failed to create MQTT publisher")
	t.Cleanup(func() {
		publisher.Disconnect(250) // Disconnect client cleanly
	})

	require.True(t, publisher.IsConnected(), "MQTT publisher is not connected")

	t.Logf("MQTT emulator test passed. Connected to Broker URL: %s", connInfo.EmulatorAddress)
}

func TestGetDefaultMqttImageContainer(t *testing.T) {
	cfg := GetDefaultMqttImageContainer()

	if cfg.EmulatorImage != mosquitoImage {
		t.Errorf("Expected image %q, got %q", mosquitoImage, cfg.EmulatorImage)
	}
	if cfg.EmulatorPort != mosquitoPort {
		t.Errorf("Expected port %q, got %q", mosquitoPort, cfg.EmulatorPort)
	}
}

func TestCreateTestMqttPublisher(t *testing.T) {
	t.Skip("Skipping TestCreateTestMqttPublisher as it relies on a running broker, tested in TestSetupMosquittoContainer")
}
