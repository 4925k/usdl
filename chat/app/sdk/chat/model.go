package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type User struct {
	ID   uuid.UUID       `json:"id"`
	Name string          `json:"name"`
	Conn *websocket.Conn `json:"-"` // do not marshal connection
}

type outMessage struct {
	From    User   `json:"from"`
	To      User   `json:"to"`
	Message string `json:"message"`
}

type inMessage struct {
	FromID  uuid.UUID `json:"fromID"`
	ToID    uuid.UUID `json:"toID"`
	Message string    `json:"message"`
}
