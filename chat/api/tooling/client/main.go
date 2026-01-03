package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	if err := connect(); err != nil {
		fmt.Println("Error:", err)
	}
}

func connect() error {
	// Connect to the WebSocket server

	url := "ws://localhost:3000/connect"
	req := http.Header{}

	socket, _, err := websocket.DefaultDialer.Dial(url, req)
	if err != nil {
		return err
	}
	defer socket.Close()

	// ------------------------------------------------------------
	// Read HELLO message from server

	_, msg, err := socket.ReadMessage()
	if err != nil {
		return err
	}

	if string(msg) != "HELLO" {
		return err
	}

	// ------------------------------------------------------------
	// Send user information to server

	user := struct {
		ID   uuid.UUID
		Name string
	}{
		ID:   uuid.New(),
		Name: "muffin",
	}

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	if err := socket.WriteMessage(websocket.TextMessage, data); err != nil {
		return err
	}

	// ------------------------------------------------------------
	// read a response from the server

	_, msg, err = socket.ReadMessage()
	if err != nil {
		return err
	}

	fmt.Println("Received message from server:", string(msg))

	return nil
}
