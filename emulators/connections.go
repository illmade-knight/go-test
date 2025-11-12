package emulators

import (
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Endpoint holds the port and full endpoint string for a service.
type Endpoint struct {
	// Port is the *internal* port of the service (e.g., "9050").
	Port string
	// Endpoint is the full, *external*, mapped endpoint (e.g., "http://localhost:32768").
	Endpoint string
}

// EmulatorConnectionInfo holds all connection details for a test emulator.
// Different fields are populated depending on the service.
type EmulatorConnectionInfo struct {
	// HTTPEndpoint is the HTTP/REST endpoint, used by GCS, BigQuery, Pub/Sub, etc.
	HTTPEndpoint Endpoint
	// GRPCEndpoint is the gRPC endpoint, primarily used by BigQuery.
	GRPCEndpoint Endpoint
	// EmulatorAddress is a generic address string for non-HTTP services
	// like MQTT ("tcp://localhost:1883") or Redis ("localhost:6379").
	EmulatorAddress string
	// ClientOptions are pre-configured Google Cloud client options
	// for connecting to the emulator (e.g., WithEndpoint, WithoutAuthentication).
	ClientOptions []option.ClientOption
}

// getEmulatorOptions returns a standard set of gRPC client options
// required to connect to Google Cloud emulators.
func getEmulatorOptions(endpoint string) []option.ClientOption {
	return []option.ClientOption{
		// Tell the client to use the emulator's HTTP/gRPC endpoint.
		option.WithEndpoint(endpoint),
		// Disable authentication, as emulators don't require it.
		option.WithoutAuthentication(),
		// Use insecure gRPC credentials, as emulators don't use TLS.
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}
}
