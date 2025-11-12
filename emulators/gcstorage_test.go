package emulators

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require" // Using require for fatal assertions
)

func TestSetupGCSEmulator(t *testing.T) {

	// Use a context with timeout for *test operations*, not container lifecycle.
	testCtx, testCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(testCancel)

	projectID := "test-project-gcs"
	baseBucket := "test-bucket"

	cfg := GetDefaultGCSConfig(projectID, baseBucket)

	// --- Setup GCS Emulator ---
	// This only starts the container and sets the env var.
	connInfo := SetupGCSEmulator(t, context.Background(), cfg)

	// --- Verify EmulatorConnectionInfo ---
	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint is empty")
	require.NotEmpty(t, connInfo.HTTPEndpoint.Port, "HTTPEndpoint.Port is empty")
	require.NotEmpty(t, connInfo.ClientOptions, "ClientOptions are empty")

	// --- Test Connectivity (using NewStorageClient) ---
	// Use testCtx for GCS client operations
	gcsClient := NewStorageClient(t, testCtx, connInfo.ClientOptions)
	require.NotNil(t, gcsClient, "NewStorageClient returned nil client")

	// --- REFACTOR ---
	// This logic is now part of the test, not the setup function.
	t.Logf("Creating test bucket: %s", baseBucket)
	err := gcsClient.Bucket(baseBucket).Create(testCtx, projectID, nil)
	require.NoError(t, err, "Failed to create base bucket %q", baseBucket)
	// --- End Refactor ---

	// Verify the base bucket exists
	_, err = gcsClient.Bucket(baseBucket).Attrs(testCtx) // Use testCtx for bucket operations
	require.NoError(t, err, "Failed to get attributes for base bucket %q", baseBucket)

	t.Logf("GCS emulator test passed. Connected to: %s", connInfo.HTTPEndpoint.Endpoint)
}

func TestGetDefaultGCSConfig(t *testing.T) {
	projectID := "default-gcs-proj"
	baseBucket := "default-bucket-name"

	cfg := GetDefaultGCSConfig(projectID, baseBucket)

	if cfg.EmulatorImage != testGCSImage {
		t.Errorf("Expected image %q, got %q", testGCSImage, cfg.EmulatorImage)
	}
	if cfg.EmulatorPort != testGCSPort {
		t.Errorf("Expected port %q, got %q", testGCSPort, cfg.EmulatorPort)
	}
	if cfg.ProjectID != projectID {
		t.Errorf("Expected project ID %q, got %q", projectID, cfg.ProjectID)
	}
	if cfg.BaseBucket != baseBucket {
		t.Errorf("Expected base bucket %q, got %q", baseBucket, cfg.BaseBucket)
	}
	if cfg.BaseStorage != "/storage/v1/b" {
		t.Errorf("Expected BaseStorage %q, got %q", "/storage/v1/b", cfg.BaseStorage)
	}
}
