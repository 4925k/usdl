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

var ErrFromNotExists = fmt.Errorf("from user does not exist")
var ErrToNotExists = fmt.Errorf("to user does not exist")

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

func (c *Chat) Handshake(ctx context.Context, conn *websocket.Conn) error {
	err := conn.WriteMessage(websocket.TextMessage, []byte("HELLO"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	msg, err := c.readMessage(ctx, conn)
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}
	var usr user
	if err := json.Unmarshal(msg, &usr); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	if err := c.addUser(usr, conn); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("already connected")); err != nil {
			return fmt.Errorf("write message: %w", err)
		}

		return fmt.Errorf("add user: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("WELCOME "+usr.Name)); err != nil {
		return err
	}

	c.log.Info(ctx, "handshake complete", "user", usr)

	return nil
}

// Find retrieves the connection for a given user.
// Returns an error if the user does not exist.
func (c *Chat) SendMessage(msg Message) error {
	c.wg.RLock()
	defer c.wg.RUnlock()

	from, ok := c.users[msg.FromID]
	if !ok {
		return ErrFromNotExists
	}

	to, ok := c.users[msg.ToID]
	if !ok {
		return ErrToNotExists
	}

	m := message{
		From: user{
			ID:   from.id,
			Name: from.name,
		},
		To: user{
			ID:   to.id,
			Name: to.name,
		},
		Message: msg.Message,
	}

	return c.send(to, m)
}

// ----------------------------------------------------------------------------------

// addUser adds a user and their connection to the chat.
// Returns an error if the user already exists.
func (c *Chat) addUser(usr user, conn *websocket.Conn) error {
	c.wg.Lock()
	defer c.wg.Unlock()

	if _, ok := c.users[usr.ID]; ok {
		return fmt.Errorf("user %s: already exists", usr.Name)
	}

	c.users[usr.ID] = connection{
		conn: conn,
		id:   usr.ID,
		name: usr.Name,
	}

	return nil
}

// removeUser removes a user from the chat.
func (c *Chat) removeUser(userID uuid.UUID) {
	c.wg.Lock()
	defer c.wg.Unlock()

	connection, ok := c.users[userID]
	if !ok {
		return
	}

	delete(c.users, userID)
	connection.conn.Close()
}

func (c *Chat) send(to connection, msg message) error {
	if err := to.conn.WriteJSON(msg); err != nil {
		c.removeUser(msg.To.ID)
		return fmt.Errorf("write json: %w", err)
	}

	return nil
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
