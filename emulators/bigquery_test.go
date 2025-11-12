package emulators

import (
	"context"
	"reflect" // Added for reflect.DeepEqual in TestGetDefaultBigQueryConfig
	"strings" // Added for checking "Already Exists" errors
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/require" // Using require for fatal assertions
)

func TestSetupBigQueryEmulator(t *testing.T) {
	t.Parallel() // Allow tests to run in parallel

	testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(testCancel)

	projectID := "test-project-bigquery"
	datasetName := "test_dataset"
	tableName := "test_table"

	type TestData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	cfg := GetDefaultBigQueryConfig(projectID, map[string]string{datasetName: tableName}, map[string]interface{}{tableName: TestData{}})

	// --- Setup BigQuery Emulator ---
	// This now *only* starts the container.
	connInfo := SetupBigQueryEmulator(t, context.Background(), cfg)

	// --- Verify EmulatorConnectionInfo ---
	require.NotEmpty(t, connInfo.HTTPEndpoint.Endpoint, "HTTPEndpoint.Endpoint is empty")
	require.NotEmpty(t, connInfo.GRPCEndpoint.Endpoint, "GRPCEndpoint.Endpoint is empty")
	require.NotEmpty(t, connInfo.HTTPEndpoint.Port, "HTTPEndpoint.Port is empty")
	require.NotEmpty(t, connInfo.GRPCEndpoint.Port, "GRPCEndpoint.Port is empty")
	require.NotEmpty(t, connInfo.ClientOptions, "ClientOptions are empty")

	// --- Test Connectivity & Resource Creation ---
	client, err := bigquery.NewClient(testCtx, projectID, connInfo.ClientOptions...)
	require.NoError(t, err, "Failed to create BigQuery client")
	t.Cleanup(func() {
		_ = client.Close()
	})

	// --- REFACTOR ---
	// This logic is now part of the test, not the setup function.
	t.Log("Creating test dataset and table...")
	for k, v := range cfg.DatasetTables {
		err = client.Dataset(k).Create(testCtx, &bigquery.DatasetMetadata{Name: k})
		if err != nil && !strings.Contains(err.Error(), "Already Exists") {
			require.NoError(t, err, "Failed to create dataset")
		}

		table := client.Dataset(k).Table(v)
		schemaType, ok := cfg.Schemas[v]
		require.True(t, ok, "Schema not found for table %s", v)
		schema, err := bigquery.InferSchema(schemaType)
		require.NoError(t, err, "Failed to infer schema")
		err = table.Create(testCtx, &bigquery.TableMetadata{Name: v, Schema: schema})
		if err != nil && !strings.Contains(err.Error(), "Already Exists") {
			require.NoError(t, err, "Failed to create table")
		}
	}
	// --- End Refactor ---

	// Verify dataset and table exist
	ds := client.Dataset(datasetName)
	_, err = ds.Metadata(testCtx) // Use testCtx for metadata operations
	require.NoError(t, err, "Failed to get dataset %q metadata", datasetName)

	table := ds.Table(tableName)
	_, err = table.Metadata(testCtx) // Use testCtx for metadata operations
	require.NoError(t, err, "Failed to get table %q metadata", tableName)

	t.Logf("BigQuery emulator test passed. Connected to HTTP: %s, gRPC: %s", connInfo.HTTPEndpoint.Endpoint, connInfo.GRPCEndpoint.Endpoint)
}

func TestGetDefaultBigQueryConfig(t *testing.T) {
	projectID := "test-proj-defaults"
	datasetTables := map[string]string{"ds1": "tbl1"}
	schemaMappings := map[string]interface{}{"tbl1": struct{ Name string }{}}

	cfg := GetDefaultBigQueryConfig(projectID, datasetTables, schemaMappings)

	if cfg.EmulatorImage != testBigQueryEmulatorImage {
		t.Errorf("Expected image %q, got %q", testBigQueryEmulatorImage, cfg.EmulatorImage)
	}
	if cfg.EmulatorPort != testBigQueryRestPort {
		t.Errorf("Expected REST port %q, got %q", testBigQueryRestPort, cfg.EmulatorPort)
	}
	if cfg.EmulatorGRPCPort != testBigQueryGRPCPort {
		t.Errorf("Expected gRPC port %q, got %q", testBigQueryGRPCPort, cfg.EmulatorGRPCPort)
	}
	if cfg.ProjectID != projectID {
		t.Errorf("Expected project ID %q, got %q", projectID, cfg.ProjectID)
	}
	if !reflect.DeepEqual(cfg.DatasetTables, datasetTables) {
		t.Errorf("DatasetTables mismatch: expected %v, got %v", datasetTables, cfg.DatasetTables)
	}
	if !reflect.DeepEqual(cfg.Schemas, schemaMappings) {
		t.Errorf("Schemas mismatch: expected %v, got %v", schemaMappings, cfg.Schemas)
	}
	if cfg.SetEnvVariables != false {
		t.Errorf("Expected SetEnvVariables to be false, got true")
	}
}
