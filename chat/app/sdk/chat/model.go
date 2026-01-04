package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type connection struct {
	conn *websocket.Conn
	id   uuid.UUID
	name string
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type message struct {
	From    user   `json:"from"`
	To      user   `json:"to"`
	Message string `json:"message"`
}

type Message struct {
	FromID  uuid.UUID `json:"from_id"`
	ToID    uuid.UUID `json:"to_id"`
	Message string    `json:"message"`
}
