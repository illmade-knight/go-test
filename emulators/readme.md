## **Refactored: readme.md (Final File)**

Markdown

# **Go Test Emulators Package**

This package provides a set of convenient helpers for spinning up service emulators for integration testing in Go.  
It's built on top of [testcontainers-go](https://github.com/testcontainers/testcontainers-go) and offers a simplified,  
consistent API for various services.

The main goal is to make it trivial to write hermetic integration tests that depend on external services like Google Cloud Pub/Sub, Firestore, BigQuery, or MQTT without needing to mock them.

## **Features ✨**

This library provides ready-to-use test helpers for the following services:

* **Google Cloud Pub/Sub**  
* **Google Cloud Firestore**  
* **Google Cloud Storage (GCS)**  
* **Google Cloud BigQuery**  
* **MQTT (Eclipse Mosquitto)**  
* **Redis**

## **Core Concepts**

The package is designed around a few simple, recurring patterns to provide a consistent developer experience across all emulators.

### 1. Configuration (GetDefault...Config)

Each emulator has a `Config` struct (e.g., `PubsubConfig`, `GCSConfig`) that holds its configuration. A helper function is provided to get a default, working configuration.

**Example**:  
`cfg := emulators.GetDefaultFirestoreConfig("my-test-project")`

### **2. Setup (Setup...Emulator or Setup...Container)**

The core of the package is the `Setup...` functions (e.g., `SetupPubsubEmulator`, `SetupRedisContainer`). These functions handle everything needed to get an emulator running for your test:

* Pulls the correct Docker image.  
* Starts the container with the right ports and command-line arguments.  
* Waits for the container to be ready to accept connections.  
* **Automatically registers a `t.Cleanup` hook** to terminate the container when your test finishes.

### **3. Connection Info (EmulatorConnectionInfo)**

All `Setup...` functions return a standardized `EmulatorConnectionInfo` struct. This provides all the necessary details to connect your Go client library to the running emulator.

````go  
// EmulatorConnectionInfo holds all the connection details for an emulator.  
type EmulatorConnectionInfo struct {  
	// HTTPEndpoint is the HTTP/REST endpoint (e.g., "http://localhost:54321")  
	HTTPEndpoint Endpoint  
	// GRPCEndpoint is the gRPC endpoint (e.g., "grpc://localhost:54322")  
	GRPCEndpoint Endpoint  
	// EmulatorAddress is for non-gRPC/HTTP services (e.g., "localhost:6379")  
	EmulatorAddress string  
	// ClientOptions are pre-configured options for Google Cloud clients  
	ClientOptions []option.ClientOption  
}
````
## **Usage Examples**

Below are examples of how to use each of the supported emulators in your Go tests.

### **Prerequisites**

Ensure you have **Docker installed and running** on your machine.

---

### **Google Cloud Pub/Sub**

The v2 Pub/Sub emulator auto-creates topics and subscriptions on first use.

Go
````
import (  
	"context"  
	"testing"  
	"cloud.google.com/go/pubsub/v2"  
	"github.com/stretchr/testify/require"  
)

func TestPubsubFeature(t *testing.T) {  
	// 1. Get the default config  
	projectID := "test-project-pubsub"  
	cfg := emulators.GetDefaultPubsubConfig(projectID)

	// 2. Start the emulator  
	// Container teardown is handled automatically.  
	ctx := context.Background()  
	connInfo := emulators.SetupPubsubEmulator(t, ctx, cfg)

	// 3. Create a client connected to the emulator  
	client, err := pubsub.NewClient(ctx, projectID, connInfo.ClientOptions...)  
	require.NoError(t, err)  
	defer client.Close()

	// 4. Use the client in your test  
	// The emulator will create the topic on first use.  
	topic := client.Topic("my-topic")  
	res := topic.Publish(ctx, &pubsub.Message{Data: []byte("hello")})  
	_, err = res.Get(ctx)  
	require.NoError(t, err)

	t.Log("Successfully connected to Pub/Sub emulator!")  
}
````
---

### **Google Cloud Storage (GCS)**

The Setup function only starts the container. The **test is responsible** for creating its own buckets.

Go
````
import (  
	"context"  
	"testing"  
	"cloud.google.com/go/storage" 
	"github.com/stretchr/testify/require"  
)

func TestGCSFeature(t *testing.T) {  
	// 1. Get the default config  
	projectID := "test-project-gcs"  
	bucketName := "my-test-bucket"  
	cfg := emulators.GetDefaultGCSConfig(projectID, bucketName)

	// 2. Start the emulator  
	ctx := context.Background()  
	connInfo := emulators.SetupGCSEmulator(t, context.Background(), cfg)

	// 3. Create a client connected to the emulator  
	// NewStorageClient auto-adds t.Cleanup for client.Close()  
	client := emulators.NewStorageClient(t, ctx, connInfo.ClientOptions)

	// 4. Create your test resources  
	err := client.Bucket(bucketName).Create(ctx, projectID, nil)  
	require.NoError(t, err, "Bucket should be created by the test")

	// 5. Use the client in your test  
	_, err = client.Bucket(bucketName).Attrs(ctx)  
	require.NoError(t, err)

	t.Log("Successfully connected to GCS emulator!")  
}
````
---

### **Google Cloud BigQuery**

The Setup function only starts the container. The **test is responsible** for creating its own datasets and tables.

Go
````
import (  
	"context"  
	"testing"  
	"cloud.google.com/go/bigquery" 
	"github.com/stretchr/testify/require"
)

func TestBigQueryFeature(t *testing.T) {  
	// 1. Define a schema and get the default config  
	type MySchema struct {  
		Name string bigquery:"name"  
	}  
	projectID := "test-project-bq"  
	datasetName := "my_dataset"  
	tableName := "my_table"  
	cfg := emulators.GetDefaultBigQueryConfig(  
		projectID,  
		map[string]string{datasetName: tableName},  
		map[string]interface{}{tableName: MySchema{}},  
	)

	// 2. Start the emulator  
	ctx := context.Background()  
	connInfo := emulators.SetupBigQueryEmulator(t, ctx, cfg)

	// 3. Create a client connected to the emulator  
	client, err := bigquery.NewClient(ctx, projectID, connInfo.ClientOptions...)  
	require.NoError(t, err)  
	defer client.Close()

	// 4. Create your test resources  
	err = client.Dataset(datasetName).Create(ctx, &bigquery.DatasetMetadata{Name: datasetName})  
	require.NoError(t, err)  
	schema, _ := bigquery.InferSchema(MySchema{})  
	err = client.Dataset(datasetName).Table(tableName).Create(ctx, &bigquery.TableMetadata{Schema: schema})  
	require.NoError(t, err)

	// 5. Use the client in your test  
	_, err = client.Dataset(datasetName).Table(tableName).Metadata(ctx)  
	require.NoError(t, err, "Table should exist")

	t.Log("Successfully connected to BigQuery emulator!")  
}
````
---

### **Google Cloud Firestore**

Go
````
import (  
	"context"  
	"testing"  
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/require"  
)

func TestFirestoreFeature(t *testing.T) {  
	// 1. Get the default config  
	projectID := "test-project-firestore"  
	cfg := emulators.GetDefaultFirestoreConfig(projectID)

	// 2. Start the emulator  
	ctx := context.Background()  
	connInfo := emulators.SetupFirestoreEmulator(t, ctx, cfg)

	// 3. Create a client connected to the emulator  
	client, err := firestore.NewClient(ctx, projectID, connInfo.ClientOptions...)  
	require.NoError(t, err)  
	defer client.Close()

	// 4. Use the client in your test  
	_, _, err = client.Collection("users").Add(ctx, map[string]interface{}{"name": "Ada"})  
	require.NoError(t, err)

	t.Log("Successfully connected to Firestore emulator!")  
}
````
---

### **Redis**

Go
````
import (  
	"context"  
	"testing"  
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"  
)

func TestRedisFeature(t *testing.T) {  
	// 1. Get the default config  
	cfg := emulators.GetDefaultRedisImageContainer()

	// 2. Start the emulator  
	connInfo := emulators.SetupRedisContainer(t, context.Background(), cfg)

	// 3. Create a client connected to the emulator  
	rdb := redis.NewClient(&redis.Options{  
		Addr: connInfo.EmulatorAddress,  
	})  
	defer rdb.Close()

	// 4. Use the client in your test  
	ctx := context.Background()  
	pong, err := rdb.Ping(ctx).Result()  
	require.NoError(t, err)  
	require.Equal(t, "PONG", pong)

	t.Log("Successfully connected to Redis emulator!")  
}
````
---

### **MQTT (Eclipse Mosquitto)**

Go
````
import (  
	"context"  
	"testing"  
	mqtt "github.com/eclipse/paho.mqtt.golang"  
	"github.com/stretchr/testify/require"  
)

func TestMqttFeature(t *testing.T) {  
	// 1. Get the default config  
	cfg := emulators.GetDefaultMqttImageContainer()

	// 2. Start the emulator  
	connInfo := emulators.SetupMosquittoContainer(t, context.Background(), cfg)  
	require.NotEmpty(t, connInfo.EmulatorAddress)

	// 3. Create a client connected to the emulator  
	publisher, err := emulators.CreateTestMqttPublisher(connInfo.EmulatorAddress, "my-publisher")  
	require.NoError(t, err)  
	defer publisher.Disconnect(250)

	// 4. Use the client in your test  
	token := publisher.Publish("test/topic", 0, false, "hello world")  
	token.Wait()  
	require.NoError(t, token.Error())

	t.Log("Successfully connected to Mosquitto emulator!")  
}  
````