package emulators

import (
	"context"
	"fmt"
	"testing"

	// REFACTOR: Use the v2 pubsub import path.
	"cloud.google.com/go/pubsub/v2"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// testEmulatorImage is the shared Google Cloud SDK emulator image for various services.
	testEmulatorImage = "gcr.io/google.com/cloudsdktool/cloud-sdk:emulators"

	// testPubsubEmulatorPort is the default port for the Pub/Sub emulator.
	testPubsubEmulatorPort = "8085"

	// testFirestoreEmulatorPort is the default port for the Firestore emulator.
	testFirestoreEmulatorPort = "8080"
)

// PubsubConfig holds configuration specific to the Pub/Sub emulator.
type PubsubConfig struct {
	GCImageContainer
	// REFACTOR: TopicSubs map is no longer needed as the v2 emulator auto-creates resources.
}

// FirestoreConfig holds configuration specific to the Firestore emulator.
type FirestoreConfig struct {
	GCImageContainer
}

// GetDefaultPubsubConfig provides a default configuration for the Pub/Sub emulator.
func GetDefaultPubsubConfig(projectID string) PubsubConfig {
	return PubsubConfig{
		GCImageContainer: GCImageContainer{
			ImageContainer: ImageContainer{
				EmulatorImage: testEmulatorImage,
				EmulatorPort:  testPubsubEmulatorPort,
			},
			ProjectID: projectID,
		},
	}
}

// GetDefaultFirestoreConfig provides a default configuration for the Firestore emulator.
func GetDefaultFirestoreConfig(projectID string) FirestoreConfig {
	return FirestoreConfig{
		GCImageContainer: GCImageContainer{
			ImageContainer: ImageContainer{
				EmulatorImage: testEmulatorImage,
				EmulatorPort:  testFirestoreEmulatorPort,
			},
			ProjectID: projectID,
		},
	}
}

// SetupPubsubEmulator starts a Pub/Sub emulator container and configures it.
// It returns client options for connecting to the emulator.
// REFACTOR: Removed the v1 admin client and resource creation logic. The emulator
// will create topics and subscriptions on first use.
func SetupPubsubEmulator(t *testing.T, ctx context.Context, cfg PubsubConfig) EmulatorConnectionInfo {
	t.Helper()

	httpPort := fmt.Sprintf("%s/tcp", cfg.EmulatorPort)
	req := testcontainers.ContainerRequest{
		Image:        cfg.EmulatorImage,
		ExposedPorts: []string{httpPort},
		Cmd:          []string{"gcloud", "beta", "emulators", "pubsub", "start", fmt.Sprintf("--project=%s", cfg.ProjectID), fmt.Sprintf("--host-port=0.0.0.0:%s", cfg.EmulatorPort)},
		WaitingFor:   wait.ForListeningPort(nat.Port(cfg.EmulatorPort)),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Failed to terminate Pub/Sub emulator container")
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, nat.Port(cfg.EmulatorPort))
	require.NoError(t, err)
	emulatorHost := fmt.Sprintf("%s:%s", host, port.Port())

	t.Logf("Pub/Sub emulator container started, listening on: %s", emulatorHost)

	clientOptions := getEmulatorOptions(emulatorHost)

	// Verify connectivity by creating a client.
	adminClient, err := pubsub.NewClient(ctx, cfg.ProjectID, clientOptions...)
	require.NoError(t, err)
	// The client is closed by the test that calls this setup function.
	t.Cleanup(func() {
		_ = adminClient.Close()
	})

	return EmulatorConnectionInfo{
		HTTPEndpoint: Endpoint{
			Port:     cfg.EmulatorPort,
			Endpoint: emulatorHost,
		},
		ClientOptions: clientOptions,
	}
}

// SetupFirestoreEmulator starts a Firestore emulator container and configures it.
func SetupFirestoreEmulator(t *testing.T, ctx context.Context, cfg FirestoreConfig) EmulatorConnectionInfo {
	t.Helper()

	httpPort := fmt.Sprintf("%s/tcp", cfg.EmulatorPort)
	req := testcontainers.ContainerRequest{
		Image:        cfg.EmulatorImage,
		ExposedPorts: []string{httpPort},
		Cmd:          []string{"gcloud", "beta", "emulators", "firestore", "start", fmt.Sprintf("--project=%s", cfg.ProjectID), fmt.Sprintf("--host-port=0.0.0.0:%s", cfg.EmulatorPort)},
		WaitingFor:   wait.ForListeningPort(nat.Port(cfg.EmulatorPort)),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Failed to terminate Firestore emulator container")
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, nat.Port(cfg.EmulatorPort))
	require.NoError(t, err)
	emulatorHost := fmt.Sprintf("%s:%s", host, port.Port())

	t.Logf("Firestore emulator container started, listening on: %s", emulatorHost)

	clientOptions := getEmulatorOptions(emulatorHost)

	return EmulatorConnectionInfo{
		HTTPEndpoint: Endpoint{
			Port:     cfg.EmulatorPort,
			Endpoint: emulatorHost,
		},
		ClientOptions: clientOptions,
	}
}
