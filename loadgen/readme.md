# **Go Load Generation Package**

This package provides a flexible and extensible framework for generating load against message-based systems. It is designed to simulate a large number of concurrent devices, each sending messages at a specified rate, making it ideal for performance and scalability testing of IoT backends, data ingestion pipelines, and other event-driven services.

## **Features 🚀**

* **Rate-Based Load Generation**: Simulate thousands of devices, each publishing messages at a specific rate (e.g., 10 messages/sec).
* **Protocol Agnostic**: The core generator is decoupled from the underlying communication protocol via a Client interface. An MqttClient is provided out of the box.
* **Customizable Payloads**: Define your own message content by implementing the PayloadGenerator interface.
* **Deterministic Simulation**: The generator's scheduling is deterministic, allowing you to calculate the exact number of expected messages for a given duration.
* **Replay Functionality**: Use the ReplayPayloadGenerator to replay a sequence of pre-recorded messages, perfect for simulating real-world scenarios.

## **Core Concepts**

The package is built around a few key interfaces and structs that work together to create a load test.

### **LoadGenerator**

This is the main orchestrator. You configure it with a set of simulated devices and a client, and then Run it for a specified duration. It manages the goroutines for each device and aggregates the results.

### **Device**

A Device struct represents a single simulated entity in your test. It has an ID, a MessageRate (in messages per second), and a PayloadGenerator.

### **Client Interface**

This interface abstracts the communication protocol. It defines three simple methods: Connect, Disconnect, and Publish. You can easily add support for other protocols (like HTTP, gRPC, or Pub/Sub) by creating a new type that implements this interface.

// Client defines the interface for a client that can publish messages.  
type Client interface {  
Connect() error  
Disconnect()  
Publish(ctx context.Context, device \*Device) (bool, error)  
}

### **PayloadGenerator Interface**

This interface is responsible for creating the content of each message. This allows you to generate random data, use stateful generators that mimic real device behavior, or replay data from a file.

// PayloadGenerator defines the interface for generating message payloads.  
type PayloadGenerator interface {  
GeneratePayload(device \*Device) (\[\]byte, error)  
}

## **Usage Example**

Here’s how to set up and run a load test using the provided MqttClient.

### **1\. Implement a PayloadGenerator**

First, create a type that generates the message payloads for your devices.

import (  
"encoding/json"  
"math/rand"  
"github.com/your/repo/loadgen" // Update with your import path  
)

// Define the structure of your message  
type TelemetryData struct {  
DeviceID    string  \`json:"device\_id"\`  
Temperature float64 \`json:"temperature"\`  
Humidity    float64 \`json:"humidity"\`  
}

// Create a generator that implements the interface  
type TelemetryGenerator struct{}

func (g \*TelemetryGenerator) GeneratePayload(device \*loadgen.Device) (\[\]byte, error) {  
payload := TelemetryData{  
DeviceID:    device.ID,  
Temperature: 20.0 \+ rand.Float64()\*10.0, // 20-30°C  
Humidity:    50.0 \+ rand.Float64()\*15.0, // 50-65%  
}  
return json.Marshal(payload)  
}

### **2\. Configure and Run the LoadGenerator**

In your test or main application, assemble the components and run the test.

import (  
"context"  
"testing"  
"time"  
"github.com/your/repo/loadgen" // Update with your import path  
"github.com/rs/zerolog"  
)

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
        {ID: "device-003", MessageRate: 2, PayloadGenerator: \&TelemetryGenerator{}},  
    }

    // 3\. Create and run the load generator  
    lg := loadgen.NewLoadGenerator(client, devices, logger)  
    duration := 30 \* time.Second

    // You can calculate the expected message count beforehand  
    expected := lg.ExpectedMessagesForDuration(duration)  
    t.Logf("Expecting approximately %d messages", expected)

    publishedCount, err := lg.Run(context.Background(), duration)  
    require.NoError(t, err)

    t.Logf("Load test finished. Successfully published %d messages.", publishedCount)  
}

### **Replaying Existing Data**

If you have a slice of byte slices (\[\]\[\]byte) representing captured messages, you can use the ReplayPayloadGenerator to publish them sequentially.

// Assume 'capturedMessages' is a \[\]\[\]byte slice  
replayGenerator := loadgen.NewReplayPayloadGenerator(capturedMessages)

devices := \[\]\*loadgen.Device{  
// This device will send one message from the slice per second  
{ID: "replay-device", MessageRate: 1, PayloadGenerator: replayGenerator},  
}

// ... create LoadGenerator and run ...  
