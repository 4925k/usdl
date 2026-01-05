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

type outMessage struct {
	From    user   `json:"from"`
	To      user   `json:"to"`
	Message string `json:"message"`
}

type inMessage struct {
	FromID  uuid.UUID `json:"fromID"`
	ToID    uuid.UUID `json:"toID"`
	Message string    `json:"message"`
}
