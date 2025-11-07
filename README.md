# Go TCP Client

A thread-safe, efficient TCP client library for Go with support for message sending, listening, automatic retry functionality, and context-based timeout control.

## Features

- **Thread-Safe Operations**: All methods are protected with mutexes for concurrent access
- **TCP Connection Management**: Connect and disconnect from TCP servers with status checking
- **Send Messages**: Send messages to the server and receive responses
- **Listen to Server Messages**: Continuously listen for incoming messages from the server using a goroutine
- **Retry Mechanism**: Automatically retry connections with configurable delays
- **Context Support**: Use Go contexts to control listening timeouts and cancellation
- **Read Deadlines**: Automatic timeout handling for read operations
- **Non-blocking Send**: Buffered message channel prevents blocking when sending messages
- **Error Handling**: Comprehensive error reporting with wrapped errors

## Installation

```bash
go get github.com/tjeumaster/go-tcp
```

## Usage

### Basic Connection

```go
package main

import (
	"fmt"
	"github.com/tjeumaster/go-tcp/tcp"
)

func main() {
	// Create a new client
	client := tcp.NewClient("127.0.0.1", "3000")
	
	// Connect to the server
	err := client.Connect()
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer client.Disconnect()
	
	// Send a message and get response
	response, err := client.SendMessage("Hello Server")
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
	fmt.Println("Response:", response)
}
```

### Connection with Retry

```go
import "time"

// Try to connect up to 5 times with 2 second delays between attempts
err := client.ConnectWithRetry(5, 2*time.Second)
if err != nil {
	fmt.Println("Failed to connect:", err)
	return
}
```

### Listen for Server Messages

```go
import "context"

// Create a context with 10 second timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Start listening for messages
err := client.Listen(ctx)
if err != nil {
	fmt.Println("Error starting listener:", err)
	return
}

// Receive messages
for msg := range client.Messages {
	fmt.Println("Received:", msg)
}
```

## API Reference

### NewClient(host, port string) *Client

Creates a new TCP client instance.

**Parameters:**
- `host`: Server hostname or IP address
- `port`: Server port number

**Returns:** Pointer to a new Client

### Connect() error

Connects to the TCP server.

**Returns:** Error if connection fails

### ConnectWithRetry(maxRetries int, retryDelay time.Duration) error

Connects to the TCP server with automatic retry on failure.

**Parameters:**
- `maxRetries`: Number of connection attempts
- `retryDelay`: Duration to wait between retry attempts

**Returns:** Error if all retry attempts fail

### Disconnect() error

Closes the TCP connection.

**Returns:** Error if disconnection fails

### IsConnected() bool

Checks if the client is currently connected to a server.

**Returns:** `true` if connected, `false` otherwise

### SendMessage(message string) (string, error)

Sends a message to the server and waits for a response.

**Parameters:**
- `message`: Message to send

**Returns:** Server response and error (if any)

### Listen(ctx context.Context) error

Starts listening for messages from the server in a background goroutine.

**Parameters:**
- `ctx`: Context for controlling the listening session (supports timeout and cancellation)

**Returns:** Error if listening setup fails

**Note:** Messages are received through the `client.Messages` channel (buffered, capacity 100)

## Examples

### Example: Listening with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := client.Listen(ctx)
if err != nil {
	fmt.Println("Error:", err)
	return
}

// Automatically stops listening after 30 seconds
for msg := range client.Messages {
	fmt.Println("Message:", msg)
}
```

### Example: Listening with Manual Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

err := client.Listen(ctx)
if err != nil {
	fmt.Println("Error:", err)
	return
}

// Stop listening after some condition
go func() {
	time.Sleep(10 * time.Second)
	cancel()
}()

for msg := range client.Messages {
	fmt.Println("Message:", msg)
}
```

### Example: Connection Status Checking

```go
client := tcp.NewClient("127.0.0.1", "3000")

// Check connection status
if client.IsConnected() {
	fmt.Println("Already connected")
} else {
	err := client.Connect()
	if err != nil {
		fmt.Println("Connection failed:", err)
		return
	}
	fmt.Println("Connected successfully")
}
```

## Architecture Notes

- **Thread-Safe Operations**: All connection operations are protected with read-write mutexes
- **Buffered Channel**: The `Messages` channel has a capacity of 100 to prevent blocking on slow receivers
- **Read Deadlines**: Automatic timeout handling prevents indefinite blocking on read operations
- **Context Integration**: Full support for Go's context pattern for timeout and cancellation control
- **Error Wrapping**: All errors are wrapped with additional context using `fmt.Errorf`

## Constants

- `DefaultBufferSize = 8096`: Default buffer size for reading messages
- `DefaultChannelBuffer = 100`: Default capacity for the messages channel

## Error Handling

The client provides detailed error messages for:
- Connection failures (with retry information)
- Message sending errors
- Response reading errors
- Listener errors (including timeout and cancellation)
- Connection state errors

## License

MIT
