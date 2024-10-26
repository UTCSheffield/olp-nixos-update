//go:build updater
// +build updater

package main

import (
	"log"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	if os.Geteuid() != 0 {
		os.Stderr.WriteString("Error: This program must be run as root.\n")
		os.Exit(1)
	}

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:9001/updater", nil)
	if err != nil {
		log.Fatal("Error connecting to updater server:", err)
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, []byte("update"))
	if err != nil {
		log.Println("Error sending update message:", err)
	}

	log.Println("Sent update message to server. Disconnecting.")
}
