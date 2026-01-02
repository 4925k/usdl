package chatapp

import (
	"encoding/json"

	"github.com/google/uuid"
)

type status struct {
	Status string `json:"status"`
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (app status) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}
