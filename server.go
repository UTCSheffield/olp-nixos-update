//go:build server
// +build server

package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

func handleClientConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error during connection upgrade:", err)
		return
	}
	defer conn.Close()

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	log.Println("Client connected:", conn.RemoteAddr())
	conn.WriteMessage(websocket.TextMessage, []byte("Welcome to the WebSocket server!"))

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
	}

	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	log.Println("Client disconnected:", conn.RemoteAddr())
}

func handleUpdaterConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error during connection upgrade:", err)
		return
	}
	defer conn.Close()

	log.Println("Updater client connected:", conn.RemoteAddr())
	err = conn.WriteMessage(websocket.TextMessage, []byte("update"))
	if err != nil {
		log.Println("Error sending message to updater client:", err)
	}

	sendUpdateToClients("update")
	log.Println("Updater client disconnecting:", conn.RemoteAddr())
}

func sendUpdateToClients(msg string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Println("Error sending update to client:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	if os.Geteuid() != 0 {
		log.Fatal("Error: This program must be run as root.\n")
	}

	http.HandleFunc("/", handleClientConnection)
	http.HandleFunc("/updater", handleUpdaterConnection)

	log.Println("WebSocket server listening on :9001 (local)")
	if err := http.ListenAndServe(":9001", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
