package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultBufferSize    = 4096
	DefaultChannelBuffer = 100
	DefaultReadTimeout   = 5 * time.Second
)

var (
	ErrNotConnected     = errors.New("client is not connected")
	ErrAlreadyListening = errors.New("already listening")
	ErrConnectionClosed = errors.New("connection closed")
)

type Client struct {
	Host      string
	Port      string
	conn      net.Conn
	Messages  chan string
	mu        sync.RWMutex
	listening atomic.Bool
}

func NewClient(host, port string) *Client {
	return &Client{
		Host:     host,
		Port:     port,
		Messages: make(chan string, DefaultChannelBuffer),
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", net.JoinHostPort(c.Host, c.Port))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	return nil
}

func (c *Client) ConnectWithRetry(maxRetries int, retryDelay time.Duration) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := c.Connect()
		if err == nil {
			fmt.Printf("Connected successfully after %d attempt(s)\n", i+1)
			return nil
		}

		lastErr = err
		fmt.Printf("Connection attempt %d failed: %v\n", i+1, err)

		if i < maxRetries-1 {
			fmt.Printf("Retrying in %v...\n", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil

	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

func (c *Client) SendMessage(message string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(DefaultReadTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	_, err := conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (c *Client) Listen(ctx context.Context) error {
	return c.ListenWithRetry(ctx, 0, 0)
}

func (c *Client) ListenWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	// Prevent multiple Listen() calls
	if !c.listening.CompareAndSwap(false, true) {
		return ErrAlreadyListening
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		c.listening.Store(false)
		return ErrNotConnected
	}

	go func() {
		defer func() {
			c.listening.Store(false)
			close(c.Messages)
			// Recreate channel for potential future Listen() calls
			c.Messages = make(chan string, DefaultChannelBuffer)
		}()

		buffer := make([]byte, DefaultBufferSize)
		retryCount := 0

		for {
			// Check context first
			if ctx.Err() != nil {
				return
			}

			// Get connection with lock to avoid race
			c.mu.RLock()
			currentConn := c.conn
			c.mu.RUnlock()

			if currentConn == nil {
				return
			}

			// Set read deadline
			if deadline, ok := ctx.Deadline(); ok {
				if err := currentConn.SetReadDeadline(deadline); err != nil {
					fmt.Printf("Failed to set read deadline: %v\n", err)
					return
				}
			} else {
				// Use longer timeout (30s) to reduce CPU usage
				currentConn.SetReadDeadline(time.Now().Add(30 * time.Second))
			}

			n, err := currentConn.Read(buffer)

			// Check context again after blocking read
			if ctx.Err() != nil {
				return
			}

			if err != nil {
				// Check if it's just a timeout
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}

				// Connection error - attempt retry if enabled
				fmt.Printf("Listen read error: %v\n", err)

				if maxRetries > 0 && retryCount < maxRetries {
					retryCount++
					fmt.Printf("Connection lost. Attempting to reconnect (%d/%d)...\n", retryCount, maxRetries)

					// Close the old connection
					c.Disconnect()

					// Wait before retrying
					time.Sleep(retryDelay)

					// Check context before reconnecting
					if ctx.Err() != nil {
						return
					}

					// Attempt to reconnect
					err := c.Connect()
					if err != nil {
						fmt.Printf("Reconnection attempt %d failed: %v\n", retryCount, err)
						if retryCount >= maxRetries {
							fmt.Println("Max reconnection attempts reached. Stopping listener.")
							return
						}
						continue
					}

					fmt.Printf("Reconnected successfully on attempt %d\n", retryCount)
					retryCount = 0 // Reset retry count on successful reconnection
					continue
				}

				// No retry configured or max retries exceeded
				return
			}

			if n > 0 {
				// Copy buffer data to avoid reuse issues
				msg := string(buffer[:n])

				select {
				case c.Messages <- msg:
					// Message sent successfully
				case <-ctx.Done():
					return
				default:
					// Channel full - log and drop
					fmt.Println("Warning: message channel full, dropping message")
				}
			}
		}
	}()

	return nil
}

func (c *Client) IsListening() bool {
	return c.listening.Load()
}

// SendAndReceive sends a message and waits for a single response.
// This is a convenience method that sends a message and returns the first response received.
// It's useful for request-response patterns where you don't need continuous listening.
func (c *Client) SendAndReceive(message string, timeout time.Duration) (string, error) {
	// Check if Listen() is already running
	if c.IsListening() {
		return "", errors.New("cannot use SendAndReceive while Listen is active")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return "", ErrNotConnected
	}

	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(DefaultReadTimeout)); err != nil {
		return "", fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send the message
	_, err := conn.Write([]byte(message))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	// Set read deadline for response
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read the response
	buffer := make([]byte, DefaultBufferSize)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(buffer[:n]), nil
}
