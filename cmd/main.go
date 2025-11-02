package main

import (
	"fmt"

	"github.com/tjeumaster/go-tcp/tcp"
)

func main() {
	host := "127.0.0.1"
	port := "3000"
	client := tcp.NewClient(host, port)
	err := client.Connect()
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	msg, err := client.SendMessage("test")
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
	fmt.Println("Received response:", msg)


}
