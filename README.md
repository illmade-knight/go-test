# **Go Test Helpers Repository**

This repository contains a collection of powerful, reusable Go packages designed to simplify and accelerate the process of writing robust integration tests. It provides helpers for spinning up service emulators, validating GCP authentication, and generating realistic load for performance testing.

## **Overview**

The repository is organized into three main packages:

* **emulators**: Start containerized emulators for various services (GCP, Redis, MQTT) directly from your Go tests.
* **auth**: Fail-fast helpers to ensure GCP credentials and permissions are correctly configured before running integration tests.
* **loadgen**: A flexible framework for simulating thousands of concurrent devices to load-test your message-based systems.

## **emulators Package**

This package provides a set of convenient helpers for spinning up service emulators for integration testing in Go. It's built on top of the excellent [testcontainers-go](https://www.google.com/search?q=%5Bhttps://github.com/testcontainers/testcontainers-go%5D\(https://github.com/testcontainers/testcontainers-go\)) library and offers a simplified, consistent API for various services.

### **Features ✨**

* **Google Cloud Pub/Sub**
* **Google Cloud Firestore**
* **Google Cloud Storage (GCS)**
* **Google Cloud BigQuery**
* **MQTT (Eclipse Mosquitto)**
* **Redis**

### **Quick Start**

All emulators follow the same simple pattern: get a default configuration, set it up (which automatically handles the container lifecycle), and use the returned connection info to create a client.

import (  
"context"  
"testing"  
"cloud.google.com/go/pubsub"  
"github.com/your/repo/emulators" // Update with your import path  
)

func TestPubsubFeature(t \*testing.T) {  
// 1\. Get the default config for the Pub/Sub emulator  
cfg := emulators.GetDefaultPubsubConfig("my-test-project", map\[string\]string{"my-topic": "my-sub"})

    // 2\. Start the emulator (teardown is automatic)  
    connInfo := emulators.SetupPubsubEmulator(t, context.Background(), cfg)

    // 3\. Create a client connected to the emulator  
    client, err := pubsub.NewClient(context.Background(), "my-test-project", connInfo.ClientOptions...)  
    require.NoError(t, err)  
    defer client.Close()

    // 4\. Use the client in your test  
    // ...  
}

## **auth Package**

This package provides test helpers for validating Google Cloud Platform (GCP) authentication and authorization within your Go integration tests. It ensures that tests that interact with real GCP services fail fast with clear, actionable error messages if the necessary credentials or permissions are not in place.

### **Features 🚀**

* **Fail-Fast Credential Checks**: Verifies that Application Default Credentials (ADC) are configured correctly.
* **Clear Error Messages**: Provides detailed, user-friendly error messages telling the developer exactly how to fix their authentication issues.
* **Permission Validation**: Checks for specific permissions required for advanced operations, like invoking Cloud Run services.
* **Automatic Test Skipping**: Skips tests gracefully if the GCP\_PROJECT\_ID environment variable isn't set.

### **Quick Start**

Add the check to the beginning of any test that requires live GCP credentials.

import (  
"testing"  
"github.com/your/repo/auth" // Update with your import path  
)

func TestMyIntegrationWithGCP(t \*testing.T) {  
// This check will fail the test with a helpful message if ADC is not set up,  
// or skip the test if GCP\_PROJECT\_ID is not set.  
projectID := auth.CheckGCPAuth(t)

    // For tests that invoke Cloud Run, use the advanced check:  
    // projectID := auth.CheckGCPAdvancedAuth(t, true)

    // ... rest of your test logic using projectID ...  
}

## **loadgen Package**

This package provides a flexible and extensible framework for generating load against message-based systems. It is designed to simulate a large number of concurrent devices, each sending messages at a specified rate.

### **Features 🚀**

* **Rate-Based Load Generation**: Simulate thousands of devices, each publishing messages at a specific rate.
* **Protocol Agnostic**: The core generator is decoupled from the underlying communication protocol via a Client interface.
* **Customizable Payloads**: Define your own message content by implementing the PayloadGenerator interface.
* **Deterministic Simulation**: Calculate the exact number of expected messages for a given duration.
* **Replay Functionality**: Use the ReplayPayloadGenerator to replay a sequence of pre-recorded messages.

### **Quick Start**

1. **Implement a PayloadGenerator** to create your message content.
2. **Configure and Run** the LoadGenerator with a client and a list of simulated devices.

import (  
"context"  
"testing"  
"time"  
"github.com/your/repo/loadgen" // Update with your import path  
"github.com/rs/zerolog"  
)

// (PayloadGenerator implementation from previous README)

func TestMyServiceLoad(t \*testing.T) {  
logger := zerolog.New(os.Stdout)  
brokerURL := "tcp://localhost:1883" // Assumes an MQTT broker is running  
topicPattern := "devices/+/telemetry"

    // 1\. Create the client for the desired protocol  
    client := loadgen.NewMqttClient(brokerURL, topicPattern, 1, logger)

    // 2\. Define the simulated devices  
    devices := \[\]\*loadgen.Device{  
        {ID: "device-001", MessageRate: 5, PayloadGenerator: \&TelemetryGenerator{}},  
        {ID: "device-002", MessageRate: 10, PayloadGenerator: \&TelemetryGenerator{}},  
    }

    // 3\. Create and run the load generator  
    lg := loadgen.NewLoadGenerator(client, devices, logger)  
    publishedCount, err := lg.Run(context.Background(), 30 \* time.Second)  
    require.NoError(t, err)

    t.Logf("Load test finished. Successfully published %d messages.", publishedCount)  
}  
