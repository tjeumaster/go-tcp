package tcp

import (
	"context"
	"fmt"
	"net"
	"time"
)

type Client struct {
	Host        string
	Port        string
	conn        *net.Conn
	Messages    chan string
	IsConnected bool
}

func NewClient(host, port string) *Client {
	return &Client{
		Host:        host,
		Port:        port,
		conn:        nil,
		Messages:    nil,
		IsConnected: false,
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", net.JoinHostPort(c.Host, c.Port))
	if err != nil {
		return err
	}

	c.conn = &conn
	c.IsConnected = true
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
	if c.conn != nil {
		err := (*c.conn).Close()
		if err != nil {
			return err
		}
		c.conn = nil
	}
	c.IsConnected = false
	return nil
}

func (c *Client) SendMessage(message string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("client is not connected")
	}

	_, err := (*c.conn).Write([]byte(message))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	buffer := make([]byte, 8096)
	n, err := (*c.conn).Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(buffer[:n]), nil
}

func (c *Client) Listen(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("client is not connected")
	}

	c.Messages = make(chan string, 100)

	go func() {
		defer close(c.Messages)
		buffer := make([]byte, 8096)
		for {
			n, err := (*c.conn).Read(buffer)

			// Check if context is done
			if ctx.Err() != nil {
				return
			}

			if err != nil {
				fmt.Println("Listen read error:", err)
				return
			}

			if n > 0 {
				// Try to send message, but don't exit on context cancellation here
				select {
				case c.Messages <- string(buffer[:n]):
				default:
					// Channel might be full, but keep listening
				}
			}
		}
	}()

	return nil
}
