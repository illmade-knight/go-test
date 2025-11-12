package emulators

// ImageContainer holds basic, non-cloud-specific container configuration.
type ImageContainer struct {
	// EmulatorImage is the full Docker image name and tag (e.g., "redis:8.0.2-alpine").
	EmulatorImage string
	// EmulatorPort is the default *internal* port the container exposes (e.g., "6379/tcp").
	EmulatorPort string
	// EmulatorGRPCPort is the secondary *internal* gRPC port, used by services like BigQuery.
	EmulatorGRPCPort string
}

// GCImageContainer extends ImageContainer with configuration specific
// to Google Cloud emulators.
type GCImageContainer struct {
	ImageContainer
	// ProjectID is the Google Cloud Project ID to configure the emulator with.
	ProjectID string
	// SetEnvVariables determines if the setup function should set environment
	// variables (like STORAGE_EMULATOR_HOST).
	SetEnvVariables bool
}
