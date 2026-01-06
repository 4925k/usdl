package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	if err := connect(); err != nil {
		fmt.Println("Error:", err)
	}
}

func connect() error {
	userName := "muffin"
	user1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	if os.Args[1] == "1" {
		user1, user2 = user2, user1
		userName = "cookie"
	}

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
		ID:   user1,
		Name: userName,
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

	// ------------------------------------------------------------
	// go routine to print messages from server

	go func() {
		_, msg, err := socket.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}

		var outMsg outMessage
		if err := json.Unmarshal(msg, &outMsg); err != nil {
			fmt.Println("Error unmarshaling message:", err)
			return
		}

		fmt.Printf("Message from %s: %s\n", outMsg.From.Name, outMsg.Message)
	}()

	// ------------------------------------------------------------
	// send a chat message to another user

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter message: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	inMsg := inMessage{
		FromID:  user1,
		ToID:    user2,
		Message: input,
	}

	data, err = json.Marshal(inMsg)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := socket.WriteMessage(websocket.TextMessage, data); err != nil {
		return err
	}

	// ------------------------------------------------------------
	// keep the main function running
	select {}
}

type inMessage struct {
	FromID  uuid.UUID `json:"fromID"`
	ToID    uuid.UUID `json:"toID"`
	Message string    `json:"message"`
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type outMessage struct {
	From    user   `json:"from"`
	To      user   `json:"to"`
	Message string `json:"message"`
}
