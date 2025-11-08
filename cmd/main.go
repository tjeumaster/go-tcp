package main

import (
	"context"
	"fmt"
	"github.com/tjeumaster/go-tcp/tcp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host := "127.0.0.1"
	port := "3000"
	client := tcp.NewClient(host, port)
	err := client.Connect()
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	err = client.Listen(ctx)
	if err != nil {
		fmt.Println("Error listening for messages:", err)
		return
	}

	count := 0

	for msg := range client.Messages {
		fmt.Println("Received message:", msg)
		count++
		if count >= 5 {
			break
		}
	}

	fmt.Println("Finished receiving messages. Exiting.")
}
