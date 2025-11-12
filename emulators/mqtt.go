package emulators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/docker/go-connections/nat" // Added for ForListeningPort
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// mosquitoImage is the default Eclipse Mosquitto image to use.
	mosquitoImage = "eclipse-mosquitto:2.0"
	// mosquitoPort is the default internal port for the Mosquitto broker.
	mosquitoPort = "1883"
)

// GetDefaultMqttImageContainer returns a default configuration for the Mosquitto container.
func GetDefaultMqttImageContainer() ImageContainer {
	return ImageContainer{
		EmulatorImage: mosquitoImage,
		EmulatorPort:  mosquitoPort,
	}
}

// SetupMosquittoContainer starts an MQTT (Mosquitto) emulator container.
// It automatically handles container startup, configuration, and teardown via t.Cleanup.
// It returns an EmulatorConnectionInfo struct with the EmulatorAddress field populated
// (e.g., "tcp://localhost:54321").
func SetupMosquittoContainer(t *testing.T, ctx context.Context, cfg ImageContainer) EmulatorConnectionInfo {
	t.Helper()

	// Mosquitto requires a config file to allow anonymous access.
	confPath := filepath.Join(t.TempDir(), "mosquitto.conf")
	err := os.WriteFile(confPath, []byte("listener 1883\nallow_anonymous true\n"), 0644)
	require.NoError(t, err)

	port := fmt.Sprintf("%s/tcp", cfg.EmulatorPort)

	req := testcontainers.ContainerRequest{
		Image:        cfg.EmulatorImage,
		ExposedPorts: []string{port},
		// REFACTOR: Changed from brittle ForLog to robust ForListeningPort.
		WaitingFor: wait.ForListeningPort(nat.Port(port)).WithStartupTimeout(60 * time.Second),
		Files:        []testcontainers.ContainerFile{{HostFilePath: confPath, ContainerFilePath: "/mosquitto/config/mosquitto.conf"}},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate Mosquitto container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := container.MappedPort(ctx, nat.Port(port)) // Use nat.Port(port) for consistency
	require.NoError(t, err)
	brokerURL := fmt.Sprintf("tcp://%s:%s", host, mappedPort.Port())

	t.Logf("Mosquitto emulator container started, listening on: %s", brokerURL)

	return EmulatorConnectionInfo{
		EmulatorAddress: brokerURL,
	}
}

// CreateTestMqttPublisher is a helper function that creates and connects an
// MQTT client (publisher) to the specified broker URL.
// It waits up to 10 seconds to connect.
func CreateTestMqttPublisher(brokerURL, clientID string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(brokerURL).SetClientID(clientID)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		return nil, fmt.Errorf("test mqtt publisher connect error: %w", token.Error())
	}
	return client, nil
}