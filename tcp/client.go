package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	DefaultBufferSize = 8096
	DefaultChannelBuffer = 100
)

type Client struct {
	Host     string
	Port     string
	conn     net.Conn
	Messages chan string
	mu       sync.RWMutex
}

func NewClient(host, port string) *Client {
	return &Client{
		Host: host,
		Port: port,
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
		return nil // Already disconnected
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

func (c *Client) SendMessage(message string) (string, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	
	if conn == nil {
		return "", fmt.Errorf("client is not connected")
	}
	
	_, err := conn.Write([]byte(message))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	
	buffer := make([]byte, DefaultBufferSize)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	return string(buffer[:n]), nil
}

func (c *Client) Listen(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	
	if conn == nil {
		return fmt.Errorf("client is not connected")
	}
	
	c.Messages = make(chan string, DefaultChannelBuffer)
	
	go func() {
		defer close(c.Messages)
		
		buffer := make([]byte, DefaultBufferSize)
		for {
			// Set read deadline based on context
			if deadline, ok := ctx.Deadline(); ok {
				if err := conn.SetReadDeadline(deadline); err != nil {
					fmt.Printf("Failed to set read deadline: %v\n", err)
					return
				}
			} else {
				// Set a reasonable timeout to periodically check context
				conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			}
			
			n, err := conn.Read(buffer)
			
			// Check if context is cancelled
			if ctx.Err() != nil {
				return
			}
			
			if err != nil {
				// Check if it's just a timeout (and context is still valid)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Context still valid, keep trying
				}
				fmt.Printf("Listen read error: %v\n", err)
				return
			}
			
			if n > 0 {
				// Copy buffer data to avoid reuse bug
				msg := string(buffer[:n])
				
				select {
				case c.Messages <- msg:
					// Message sent successfully
				case <-ctx.Done():
					return
				default:
					// Channel full - you could log this or handle differently
					fmt.Println("Warning: message channel full, dropping message")
				}
			}
		}
	}()
	
	return nil
}