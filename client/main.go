package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/websocket"
)

type Config struct {
	Config string `toml:"config"`
}

func updateCommands() {
	git_pull := exec.Command("git", []string{"pull"}...)
	git_pull.Dir = "/etc/nixos"
	git_pull.Stdout = os.Stdout
	git_pull.Stderr = os.Stderr
	git_pull.Run()

	config, _ := readConfig()
	println(config)

	nixos_rebuild := exec.Command("nixos-rebuild", []string{"switch", "--flake", fmt.Sprint("/etc/nixos#", config)}...)
	nixos_rebuild.Stdout = os.Stdout
	nixos_rebuild.Stderr = os.Stderr
	nixos_rebuild.Run()
}

func readConfig() (string, error) {
	var config Config
	if _, err := toml.DecodeFile("/root/setup.toml", &config); err != nil {
		return "", fmt.Errorf("failed to read config: %v", err)
	}
	return config.Config, nil
}

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

	serverURL := "ws://127.0.0.1:9001/"

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
				if string(message) == "update" {
					updateCommands()
				}
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
