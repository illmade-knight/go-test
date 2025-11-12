package emulators

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// cloudTestRedisImage is the default Redis image to use.
	cloudTestRedisImage = "redis:8.0.2-alpine"
	// cloudTestRedisPort is the default internal port for Redis.
	cloudTestRedisPort = "6379/tcp"
)

// GetDefaultRedisImageContainer returns a default configuration for the Redis container.
func GetDefaultRedisImageContainer() ImageContainer {
	return ImageContainer{
		EmulatorImage: cloudTestRedisImage,
		EmulatorPort:  cloudTestRedisPort,
	}
}

// SetupRedisContainer starts a Redis container and returns its connection information.
// It automatically handles container startup and teardown via t.Cleanup.
// It returns an EmulatorConnectionInfo struct with the EmulatorAddress field populated
// (e.g., "localhost:54321").
func SetupRedisContainer(t *testing.T, ctx context.Context, imageContainer ImageContainer) EmulatorConnectionInfo {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        imageContainer.EmulatorImage,
		ExposedPorts: []string{imageContainer.EmulatorPort},
		WaitingFor:   wait.ForListeningPort(nat.Port(imageContainer.EmulatorPort)).WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err, "Failed to start Redis container")

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate Redis container: %v", err)
		}
		t.Log("Redis container terminated.")
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, nat.Port(imageContainer.EmulatorPort))
	require.NoError(t, err)

	redisAddr := fmt.Sprintf("%s:%s", host, port.Port())
	t.Logf("Redis container started at: %s", redisAddr)

	return EmulatorConnectionInfo{
		EmulatorAddress: redisAddr,
	}
}
