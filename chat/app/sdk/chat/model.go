package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type connection struct {
	conn *websocket.Conn
	id   uuid.UUID
	name string
}
