package emulators

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// BigQueryConfig holds configuration specific to the BigQuery emulator.
type BigQueryConfig struct {
	GCImageContainer
	// DatasetTables holds a map of dataset names to table names.
	// This is used by the *test* to know what to create, not by the setup function.
	DatasetTables map[string]string
	// Schemas holds a map of table names to their Go struct schema.
	// This is used by the *test* to infer and create the table schema.
	Schemas map[string]interface{}
}

const (
	// testBigQueryEmulatorImage is the default BigQuery emulator image.
	testBigQueryEmulatorImage = "ghcr.io/goccy/bigquery-emulator:0.6.6"
	// testBigQueryGRPCPort is the default gRPC port for the emulator.
	testBigQueryGRPCPort = "9060"
	// testBigQueryRestPort is the default REST port for the emulator.
	testBigQueryRestPort = "9050"
)

// GetDefaultBigQueryConfig provides a default configuration for the BigQuery emulator.
// The provided maps are used by the test to create resources.
func GetDefaultBigQueryConfig(projectID string, datasetTables map[string]string, schemaMappings map[string]interface{}) BigQueryConfig {
	return BigQueryConfig{
		GCImageContainer: GCImageContainer{
			ImageContainer: ImageContainer{
				EmulatorImage:    testBigQueryEmulatorImage,
				EmulatorPort:     testBigQueryRestPort,
				EmulatorGRPCPort: testBigQueryGRPCPort,
			},
			ProjectID:       projectID,
			SetEnvVariables: false,
		},
		DatasetTables: datasetTables,
		Schemas:       schemaMappings,
	}
}

// SetupBigQueryEmulator starts a BigQuery emulator container.
// It automatically handles container startup and teardown via t.Cleanup.
//
// This function *only* starts the emulator. It does NOT create any datasets or
// tables. The test calling this function is responsible for creating its own
// resources using the returned EmulatorConnectionInfo.
func SetupBigQueryEmulator(t *testing.T, ctx context.Context, cfg BigQueryConfig) EmulatorConnectionInfo {
	t.Helper()
	httpPort := fmt.Sprintf("%s/tcp", cfg.EmulatorPort)
	grpcPort := fmt.Sprintf("%s/tcp", cfg.EmulatorGRPCPort)
	req := testcontainers.ContainerRequest{
		Image:        cfg.EmulatorImage,
		ExposedPorts: []string{httpPort, grpcPort},
		Cmd: []string{
			"--project=" + cfg.ProjectID,
			"--port=" + cfg.EmulatorPort,
			"--grpc-port=" + cfg.EmulatorGRPCPort,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(nat.Port(httpPort)).WithStartupTimeout(60*time.Second),
			wait.ForListeningPort(nat.Port(grpcPort)).WithStartupTimeout(60*time.Second),
		),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate BigQuery container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	mappedGrpcPort, err := container.MappedPort(ctx, nat.Port(grpcPort))
	require.NoError(t, err)
	mappedRestPort, err := container.MappedPort(ctx, nat.Port(httpPort))
	require.NoError(t, err)

	endpointGRPC := fmt.Sprintf("grpc://%s:%s", host, mappedGrpcPort.Port())
	endpointHTTP := fmt.Sprintf("http://%s:%s", host, mappedRestPort.Port())
	opts := getEmulatorOptions(endpointHTTP)

	// --- Key Refactor ---
	// Removed the resource creation loop.
	// We verify connectivity by creating a client, but we don't
	// modify state.
	client, err := bigquery.NewClient(ctx, cfg.ProjectID, opts...)
	require.NoError(t, err)
	_ = client.Close() // Close the temporary client immediately.

	t.Logf("BigQuery emulator container started. HTTP: %s, gRPC: %s", endpointHTTP, endpointGRPC)

	return EmulatorConnectionInfo{
		HTTPEndpoint: Endpoint{
			Port:     httpPort,
			Endpoint: endpointHTTP,
		},
		GRPCEndpoint: Endpoint{
			Port:     grpcPort,
			Endpoint: endpointGRPC,
		},
		ClientOptions: opts,
	}
}
