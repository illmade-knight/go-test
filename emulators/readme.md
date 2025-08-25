# **Go Test Emulators Package**

This package provides a set of convenient helpers for spinning up service emulators for integration testing in Go. 
It's built on top of [testcontainers-go](https://www.google.com/search?q=%5Bhttps://github.com/testcontainers/testcontainers-go%5D\(https://github.com/testcontainers/testcontainers-go\)) library and offers a simplified, 
consistent API for various services.

The main goal is to make it trivial to write hermetic integration tests that depend on external services like Google Cloud Pub/Sub, Firestore, BigQuery, Redis, or MQTT without needing to mock them.

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

### **1\. Configuration (GetDefault...Config)**

Each emulator has a Config struct (e.g., PubsubConfig, RedisConfig) that holds its configuration. A helper function is provided to get a default, working configuration for each. You only need to provide test-specific details like a project ID and resource names.

**Example**:

cfg := emulators.GetDefaultPubsubConfig("my-test-project", map\[string\]string{"my-topic": "my-sub"})

### **2\. Setup (Setup...Emulator or Setup...Container)**

The core of the package is the Setup functions (e.g., SetupPubsubEmulator, SetupRedisContainer). These functions handle everything needed to get an emulator running for your test:

* Pulls the correct Docker image.
* Starts the container with the right ports and command-line arguments.
* Waits for the container to be ready.
* **Automatically registers a t.Cleanup hook** to terminate the container when your test finishes. You don't need to manage the container's lifecycle.
* Pre-creates any necessary resources (like Pub/Sub topics or BigQuery datasets).

### **3\. Connection Info (EmulatorConnectionInfo)**

All Setup functions return a standardized EmulatorConnectionInfo struct. This provides all the necessary details to connect your Go client library to the running emulator.

// EmulatorConnectionInfo holds all the connection details for an emulator.  
type EmulatorConnectionInfo struct {  
HTTPEndpoint    Endpoint  
GRPCEndpoint    Endpoint  
EmulatorAddress string                // For non-gRPC/HTTP services like MQTT or Redis  
ClientOptions   \[\]option.ClientOption // For Google Cloud clients  
}

## **Usage Examples**

Below are examples of how to use each of the supported emulators in your Go tests.

### **Prerequisites**

Ensure you have **Docker installed and running** on your machine.

### **Google Cloud Pub/Sub**

import (  
"context"  
"testing"  
"cloud.google.com/go/pubsub"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestPubsubFeature(t \*testing.T) {  
// 1\. Get the default config for the Pub/Sub emulator  
projectID := "test-project-pubsub"  
topicName := "my-topic"  
subName := "my-subscription"  
cfg := emulators.GetDefaultPubsubConfig(projectID, map\[string\]string{topicName: subName})

    // 2\. Start the emulator  
    // Container teardown is handled automatically by t.Cleanup within the function.  
    connInfo := emulators.SetupPubsubEmulator(t, context.Background(), cfg)

    // 3\. Create a client connected to the emulator  
    ctx := context.Background()  
    client, err := pubsub.NewClient(ctx, projectID, connInfo.ClientOptions...)  
    require.NoError(t, err)  
    defer client.Close()

    // 4\. Use the client in your test  
    topic := client.Topic(topicName)  
    exists, err := topic.Exists(ctx)  
    require.NoError(t, err)  
    require.True(t, exists, "Topic should have been created automatically")

    t.Log("Successfully connected to Pub/Sub emulator\!")  
}

### **Redis**

import (  
"context"  
"testing"  
"github.com/redis/go-redis/v9"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestRedisFeature(t \*testing.T) {  
// 1\. Get the default config  
cfg := emulators.GetDefaultRedisImageContainer()

    // 2\. Start the emulator  
    connInfo := emulators.SetupRedisContainer(t, context.Background(), cfg)

    // 3\. Create a client connected to the emulator  
    rdb := redis.NewClient(\&redis.Options{  
        Addr: connInfo.EmulatorAddress,  
    })  
    defer rdb.Close()

    // 4\. Use the client in your test  
    ctx := context.Background()  
    pong, err := rdb.Ping(ctx).Result()  
    require.NoError(t, err)  
    require.Equal(t, "PONG", pong)

    t.Log("Successfully connected to Redis emulator\!")  
}

### **Google Cloud Firestore**

import (  
"context"  
"testing"  
"cloud.google.com/go/firestore"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestFirestoreFeature(t \*testing.T) {  
// 1\. Get the default config  
projectID := "test-project-firestore"  
cfg := emulators.GetDefaultFirestoreConfig(projectID)

    // 2\. Start the emulator  
    connInfo := emulators.SetupFirestoreEmulator(t, context.Background(), cfg)

    // 3\. Create a client connected to the emulator  
    ctx := context.Background()  
    client, err := firestore.NewClient(ctx, projectID, connInfo.ClientOptions...)  
    require.NoError(t, err)  
    defer client.Close()

    // 4\. Use the client in your test  
    \_, \_, err \= client.Collection("users").Add(ctx, map\[string\]interface{}{"name": "Ada"})  
    require.NoError(t, err)

    t.Log("Successfully connected to Firestore emulator\!")  
}

### **Google Cloud Storage (GCS)**

import (  
"context"  
"testing"  
"cloud.google.com/go/storage"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestGCSFeature(t \*testing.T) {  
// 1\. Get the default config  
projectID := "test-project-gcs"  
bucketName := "my-test-bucket"  
cfg := emulators.GetDefaultGCSConfig(projectID, bucketName)

    // 2\. Start the emulator  
    connInfo := emulators.SetupGCSEmulator(t, context.Background(), cfg)

    // 3\. Create a client connected to the emulator  
    ctx := context.Background()  
    // You can use the helper or create the client directly  
    client := emulators.GetStorageClient(t, ctx, cfg, connInfo.ClientOptions)

    // 4\. Use the client in your test  
    \_, err := client.Bucket(bucketName).Attrs(ctx)  
    require.NoError(t, err, "Bucket should have been created automatically")

    t.Log("Successfully connected to GCS emulator\!")  
}

### **Google Cloud BigQuery**

import (  
"context"  
"testing"  
"cloud.google.com/go/bigquery"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestBigQueryFeature(t \*testing.T) {  
// 1\. Define a schema and get the default config  
type MySchema struct {  
Name string \`bigquery:"name"\`  
Count int   \`bigquery:"count"\`  
}  
projectID := "test-project-bq"  
datasetName := "my\_dataset"  
tableName := "my\_table"  
cfg := emulators.GetDefaultBigQueryConfig(  
projectID,  
map\[string\]string{datasetName: tableName},  
map\[string\]interface{}{tableName: MySchema{}},  
)

    // 2\. Start the emulator  
    connInfo := emulators.SetupBigQueryEmulator(t, context.Background(), cfg)

    // 3\. Create a client connected to the emulator  
    ctx := context.Background()  
    client, err := bigquery.NewClient(ctx, projectID, connInfo.ClientOptions...)  
    require.NoError(t, err)  
    defer client.Close()

    // 4\. Use the client in your test  
    table := client.Dataset(datasetName).Table(tableName)  
    \_, err \= table.Metadata(ctx)  
    require.NoError(t, err, "Table should have been created automatically")

    t.Log("Successfully connected to BigQuery emulator\!")  
}

### **MQTT (Eclipse Mosquitto)**

import (  
"context"  
"testing"  
mqtt "github.com/eclipse/paho.mqtt.golang"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestMqttFeature(t \*testing.T) {  
// 1\. Get the default config  
cfg := emulators.GetDefaultMqttImageContainer()

    // 2\. Start the emulator  
    connInfo := emulators.SetupMosquittoContainer(t, context.Background(), cfg)  
    require.NotEmpty(t, connInfo.EmulatorAddress)

    // 3\. Create a client connected to the emulator  
    publisher, err := emulators.CreateTestMqttPublisher(connInfo.EmulatorAddress, "my-publisher")  
    require.NoError(t, err)  
    defer publisher.Disconnect(250)

    // 4\. Use the client in your test  
    token := publisher.Publish("test/topic", 0, false, "hello world")  
    token.Wait()  
    require.NoError(t, token.Error())

    t.Log("Successfully connected to Mosquitto emulator\!")  
}  
