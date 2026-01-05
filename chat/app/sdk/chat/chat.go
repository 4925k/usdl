// Package chat provides support for chat activities.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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
	mu    sync.RWMutex
}

// NewChat creates a new chat manager.
func NewChat(log *logger.Logger) *Chat {
	c := &Chat{
		log: log,
	}

	c.ping()

	return c
}

// Handshake performs the handshake process for a new connection.
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

// Listen starts listening for messages from the connected user.
func (c *Chat) Listen(ctx context.Context, conn *websocket.Conn) {
	for {
		msg, err := c.readMessage(ctx, conn)
		if err != nil {
			c.log.Error(ctx, "read message failed", "error", err)
			return
		}

		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Error(ctx, "unmarshal message failed", "error", err)
			return
		}

		if err := c.sendMessage(inMsg); err != nil {
			c.log.Error(ctx, "send message failed", "error", err)
			continue
		}
	}

}

// ----------------------------------------------------------------------------------

// sendMessage sends a message from one user to another.
func (c *Chat) sendMessage(msg inMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	from, ok := c.users[msg.FromID]
	if !ok {
		return ErrFromNotExists
	}

	to, ok := c.users[msg.ToID]
	if !ok {
		return ErrToNotExists
	}

	m := outMessage{
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

	if err := to.conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write json: %w", err)
	}

	return nil
}

// connections returns a copy of all current connections.
func (c *Chat) connections() map[uuid.UUID]connection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conns := make(map[uuid.UUID]connection, len(c.users))
	maps.Copy(conns, c.users)

	return conns
}

// ping sends periodic ping messages to all connected users.
func (c *Chat) ping() {
	ticket := time.NewTicker(time.Second * 10)

	go func() {
		for {
			<-ticket.C

			c.log.Info(context.Background(), "PING", "status", "started")

			conns := c.connections() // copy connections to avoid read locking during ping
			for _, conn := range conns {
				c.log.Info(context.Background(), "PING", "user", conn.name, "id", conn.id)
				if err := conn.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.removeUser(conn.id) // remove user on ping failure only so that there's a single source of truth
				}
			}

			c.log.Info(context.Background(), "PING", "status", "completed")
		}
	}()
}

// addUser adds a user and their connection to the chat.
// Returns an error if the user already exists.
func (c *Chat) addUser(usr user, conn *websocket.Conn) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.users[usr.ID]; ok {
		return fmt.Errorf("user already exists")
	}

	c.users[usr.ID] = connection{
		conn: conn,
		id:   usr.ID,
		name: usr.Name,
	}

	c.log.Info(context.Background(), "added user", "user", usr.Name, "id", usr.ID)

	return nil
}

// removeUser removes a user from the chat.
func (c *Chat) removeUser(userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	connection, ok := c.users[userID]
	if !ok {
		c.log.Info(context.Background(), "remove user: user does not exist", "id", userID)
		return
	}

	c.log.Info(context.Background(), "removing user", "user", connection.name, "id", connection.id)

	delete(c.users, userID)
	connection.conn.Close()
}

// readMessage reads a message from the websocket connection with context cancellation support.
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
