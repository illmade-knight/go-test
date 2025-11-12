package emulators

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
)

const (
	// testGCSImage is the default fake-gcs-server image to use.
	testGCSImage = "fsouza/fake-gcs-server:latest"
	// testGCSPort is the default internal port for the fake-gcs-server.
	testGCSPort = "4443"
)

// GCSConfig holds configuration specific to the GCS emulator.
type GCSConfig struct {
	GCImageContainer
	// BaseBucket is the name of a bucket the *test* may want to create.
	// This is not created automatically.
	BaseBucket string
	// BaseStorage is the internal health check path for the emulator.
	BaseStorage string
}

// GetDefaultGCSConfig provides a default configuration for the GCS emulator.
func GetDefaultGCSConfig(projectID, baseBucket string) GCSConfig {
	return GCSConfig{
		GCImageContainer: GCImageContainer{
			ImageContainer: ImageContainer{
				EmulatorImage: testGCSImage,
				EmulatorPort:  testGCSPort,
			},
			ProjectID: projectID,
		},
		BaseBucket:  baseBucket,
		BaseStorage: "/storage/v1/b",
	}
}

// NewStorageClient creates a new GCS storage client configured for the emulator.
// It automatically registers a t.Cleanup hook to close the client.
//
// This function *only* creates a client. It does NOT create any buckets.
func NewStorageClient(t *testing.T, ctx context.Context, opts []option.ClientOption) *storage.Client {
	t.Helper()
	gcsClient, err := storage.NewClient(ctx, opts...)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, gcsClient.Close())
	})

	// --- Key Refactor ---
	// Removed the automatic bucket creation. The test is now responsible
	// for creating its own buckets.
	// if cfg.BaseBucket != "" {
	// 	err = gcsClient.Bucket(cfg.BaseBucket).Create(ctx, cfg.ProjectID, nil)
	// 	require.NoError(t, err)
	// }

	return gcsClient
}

// SetupGCSEmulator starts a GCS emulator (fake-gcs-server) container.
// It automatically handles container startup and teardown via t.Cleanup.
// It returns connection info for creating a client.
func SetupGCSEmulator(t *testing.T, ctx context.Context, cfg GCSConfig) EmulatorConnectionInfo {
	t.Helper()

	httpPort := fmt.Sprintf("%s/tcp", cfg.EmulatorPort)
	req := testcontainers.ContainerRequest{
		Image:        cfg.EmulatorImage,
		ExposedPorts: []string{httpPort},
		Cmd:          []string{"-scheme", "http"}, // Explicitly tell fake-gcs-server to use http
		WaitingFor: wait.ForHTTP(cfg.BaseStorage).WithPort(nat.Port(httpPort)).WithStatusCodeMatcher(
			func(status int) bool {
				// The fake-gcs-server returns 400 for an empty listing, which is healthy.
				return status > 0
			}).WithStartupTimeout(20 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate GCS container: %v", err)
		}
	})

	// The endpoint must be retrieved and set as an environment variable
	// for the GCS client library to work correctly without https.
	emulatorEndpoint, err := container.Endpoint(ctx, "") // Returns "host:port"
	require.NoError(t, err)
	t.Setenv("STORAGE_EMULATOR_HOST", emulatorEndpoint)
	t.Logf("GCS emulator container started at: %s", emulatorEndpoint)

	// Note: The GCS client options are special. They rely on the
	// STORAGE_EMULATOR_HOST env var and do not use getEmulatorOptions().
	// We return a "clean" set of options (no endpoint, no insecure credentials)
	// and let the Google client library automatically detect the env var.
	opts := []option.ClientOption{
		option.WithoutAuthentication(),
	}

	// We must also create a client here to verify connectivity
	// before the env var (t.Setenv) goes out of scope.
	client, err := storage.NewClient(ctx, opts...)
	require.NoError(t, err)
	_ = client.Close()

	return EmulatorConnectionInfo{
		HTTPEndpoint: Endpoint{
			Port:     cfg.EmulatorPort,
			Endpoint: emulatorEndpoint, // This is just "host:port"
		},
		ClientOptions: opts,
	}
}
