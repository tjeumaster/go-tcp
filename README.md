# Go TCP Client

A thread-safe, efficient TCP client library for Go with support for message sending, listening, automatic retry functionality, and context-based timeout control.

## Features

- **Thread-Safe Operations**: All methods are protected with mutexes and atomic operations for concurrent access
- **TCP Connection Management**: Connect and disconnect from TCP servers with status checking
- **Send Messages**: Send messages to the server and receive responses with automatic timeout handling
- **Listen to Server Messages**: Continuously listen for incoming messages from the server using a goroutine
- **Connection Retry**: Automatically retry connections with configurable delays
- **Auto-Reconnect on Disconnect**: Automatically reconnect when connection is lost during listening
- **Context Support**: Use Go contexts to control listening timeouts and cancellation
- **Read/Write Deadlines**: Automatic timeout handling for both read and write operations
- **Non-blocking Send**: Buffered message channel prevents blocking when sending messages
- **Error Definitions**: Pre-defined error types for better error handling
- **Listening Status**: Track whether the client is actively listening with `IsListening()`
- **Multiple Listen Prevention**: Prevents multiple concurrent `Listen()` calls
- **Smart Timeout Management**: Configurable default timeouts with context deadline support

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

### IsListening() bool

Checks if the client is currently listening for messages.

**Returns:** `true` if actively listening, `false` otherwise

### SendMessage(message string) error

Sends a message to the server with automatic timeout handling.

**Parameters:**
- `message`: Message to send

**Returns:** Error (if any)

**Note:** 
- Only sends the message; responses are received through the `Messages` channel when using `Listen()`
- Sets write deadline using `DefaultReadTimeout` (5 seconds)
- Safe to call from multiple goroutines concurrently

### SendAndReceive(message string, timeout time.Duration) (string, error)

Sends a message and waits for a single response. Useful for simple request-response patterns.

**Parameters:**
- `message`: Message to send
- `timeout`: Maximum time to wait for the response

**Returns:** Response string and error (if any)

**Note:**
- Cannot be used while `Listen()` is active (returns error if listening)
- Blocks until response is received or timeout expires
- Best for synchronous request-response patterns
- Sets both write and read deadlines

### Listen(ctx context.Context) error

Starts listening for messages from the server in a background goroutine without automatic reconnection.

**Parameters:**
- `ctx`: Context for controlling the listening session (supports timeout and cancellation)

**Returns:** Error if listening setup fails

**Note:** 
- Messages are received through the `client.Messages` channel (buffered, capacity 100)
- Only one `Listen()` call can be active at a time (returns `ErrAlreadyListening` if already listening)
- Uses context deadline if provided, otherwise applies a 30-second read timeout
- Automatically recreates the message channel when listening stops
- Connection errors will stop the listener

### ListenWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) error

Starts listening for messages with automatic reconnection on connection loss.

**Parameters:**
- `ctx`: Context for controlling the listening session (supports timeout and cancellation)
- `maxRetries`: Number of reconnection attempts when connection is lost (0 = no retry)
- `retryDelay`: Duration to wait between reconnection attempts

**Returns:** Error if listening setup fails

**Note:**
- Automatically attempts to reconnect when connection is lost
- Resets retry count after each successful reconnection
- Continues listening seamlessly after reconnection
- Stops after max retries exceeded or context cancelled

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

### Example: Listen with Automatic Reconnection

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

client := tcp.NewClient("127.0.0.1", "3000")

err := client.Connect()
if err != nil {
	fmt.Println("Error connecting:", err)
	return
}
defer client.Disconnect()

// Listen with automatic reconnection (5 retries, 2 second delay)
err = client.ListenWithRetry(ctx, 5, 2*time.Second)
if err != nil {
	fmt.Println("Error starting listener:", err)
	return
}

// Receive messages - will continue receiving even after reconnections
for msg := range client.Messages {
	fmt.Println("Received:", msg)
}
```

### Example: Send and Receive Pattern

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

client := tcp.NewClient("127.0.0.1", "3000")

err := client.Connect()
if err != nil {
	fmt.Println("Error connecting:", err)
	return
}
defer client.Disconnect()

// Start listening for responses
err = client.Listen(ctx)
if err != nil {
	fmt.Println("Error starting listener:", err)
	return
}

// Send messages from main goroutine
err = client.SendMessage("Hello, Server!")
if err != nil {
	fmt.Println("Error sending message:", err)
}

// Receive responses from listener goroutine
for msg := range client.Messages {
	fmt.Println("Server response:", msg)
	
	// Can send more messages based on responses
	if msg == "READY" {
		client.SendMessage("START")
	}
}
```

## Architecture Notes

- **Thread-Safe Operations**: All connection operations are protected with read-write mutexes
- **Atomic Listening State**: Uses `atomic.Bool` to track listening state without locks
- **Buffered Channel**: The `Messages` channel has a capacity of 100 to prevent blocking on slow receivers
- **Read/Write Deadlines**: Automatic timeout handling for both read and write operations
  - Send operations timeout after `DefaultReadTimeout` (5 seconds)
  - Listen operations use context deadline or 30-second read timeout
- **Context Integration**: Full support for Go's context pattern for timeout and cancellation control
- **Error Wrapping**: All errors are wrapped with additional context using `fmt.Errorf`
- **Connection Safety**: Double-checks connection state within goroutines to prevent race conditions
- **Channel Recreation**: Automatically recreates the message channel when listening stops, allowing multiple `Listen()` calls in sequence
- **Automatic Reconnection**: `ListenWithRetry()` automatically reconnects on connection loss with configurable retry logic
- **Separate Send/Receive**: `SendMessage()` only sends; all responses come through the `Messages` channel for clean concurrency

## Constants

- `DefaultBufferSize = 8096`: Default buffer size for reading messages
- `DefaultChannelBuffer = 100`: Default capacity for the messages channel
- `DefaultReadTimeout = 5 seconds`: Default timeout for send/receive operations

## Error Types

Pre-defined error variables for better error handling:

- `ErrNotConnected`: Returned when operations are attempted without an active connection
- `ErrAlreadyListening`: Returned when `Listen()` is called while already listening
- `ErrConnectionClosed`: Returned when the connection is unexpectedly closed

## Error Handling

The client provides detailed error messages and predefined error types:

- **ErrNotConnected**: Connection required but not established
- **ErrAlreadyListening**: Multiple concurrent listen attempts
- **ErrConnectionClosed**: Connection unexpectedly closed
- Custom wrapped errors: All errors are wrapped with operation context for debugging

### Example: Error Handling

```go
import "errors"

err := client.SendMessage("test")
if errors.Is(err, tcp.ErrNotConnected) {
	fmt.Println("Need to connect first")
} else if err != nil {
	fmt.Println("Send failed:", err)
}
```

## License

MIT
