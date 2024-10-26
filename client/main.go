package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func connectToServer(urlStr string) (*websocket.Conn, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func main() {
	if os.Geteuid() != 0 {
		log.Fatal("Error: This program must be run as root.")
	}

	serverURL := "ws://127.0.0.1:9001/local"

	for {
		conn, err := connectToServer(serverURL)
		if err != nil {
			log.Printf("Failed to connect to server: %v. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("Connected to server:", serverURL)

		// Listen for messages from the server
		go func() {
			defer conn.Close()
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("Disconnected from server: %v. Attempting to reconnect...\n", err)
					return // Exit the goroutine to reconnect
				}
				log.Printf("Received message: %s\n", message)
			}
		}()

		// Keep the main thread alive to maintain the connection
		for {
			// Wait for the connection to be closed
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("Connection lost. Waiting to reconnect...")
				break // Break the loop to reconnect
			}
			time.Sleep(10 * time.Second) // Keep-alive ping every 30 seconds
		}
	}
}
