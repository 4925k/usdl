// Package chat provides support for chat activities.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var ErrUserExists = fmt.Errorf("user already exists")
var ErruserNotExists = fmt.Errorf("user does not exist")

// Chat manages chat operations.
type Chat struct {
	log   *logger.Logger
	users map[uuid.UUID]connection
	wg    sync.RWMutex
}

// NewChat creates a new chat manager.
func NewChat(log *logger.Logger) *Chat {
	return &Chat{
		log: log,
	}
}

// AddUser adds a user and their connection to the chat.
// Returns an error if the user already exists.
func (c *Chat) AddUser(usr User, conn *websocket.Conn) error {
	c.wg.Lock()
	defer c.wg.Unlock()

	if _, ok := c.users[usr.ID]; ok {
		return ErrUserExists
	}

	c.users[usr.ID] = connection{
		conn: conn,
		id:   usr.ID,
		name: usr.Name,
	}

	return nil
}

// RemoveUser removes a user from the chat.
func (c *Chat) RemoveUser(userID uuid.UUID) error {
	c.wg.Lock()
	defer c.wg.Unlock()

	if _, ok := c.users[userID]; !ok {
		return ErruserNotExists
	}

	delete(c.users, userID)

	return nil
}

// Find retrieves the connection for a given user.
// Returns an error if the user does not exist.
func (c *Chat) Find(userID uuid.UUID) (User, error) {
	c.wg.RLock()
	defer c.wg.RUnlock()

	conn, ok := c.users[userID]
	if !ok {
		return User{}, ErruserNotExists
	}

	return User{
		ID:   conn.id,
		Name: conn.name,
	}, nil
}

func (c *Chat) Handshake(ctx context.Context, conn *websocket.Conn) (User, error) {
	err := conn.WriteMessage(websocket.TextMessage, []byte("HELLO"))
	if err != nil {
		return User{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	msg, err := c.readMessage(ctx, conn)
	if err != nil {
		return User{}, fmt.Errorf("read message: %w", err)
	}

	var usr User
	if err := json.Unmarshal(msg, &usr); err != nil {
		return User{}, fmt.Errorf("unmarshal message: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("WELCOME "+usr.Name)); err != nil {
		return User{}, err
	}

	return usr, nil
}

func (c *Chat) readMessage(ctx context.Context, conn *websocket.Conn) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)

	go func() {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			ch <- response{nil, err}
		}

		ch <- response{msg, nil}
	}()

	var resp response

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	case resp = <-ch:
		if resp.msg == nil {
			return nil, fmt.Errorf("empty message")
		}
	}

	return resp.msg, nil
}
