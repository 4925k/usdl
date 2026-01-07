// Package chat provides support for chat activities.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/4925k/usdl/chat/app/sdk/errs"
	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var ErrFromNotExists = fmt.Errorf("from user does not exist")
var ErrToNotExists = fmt.Errorf("to user does not exist")

// Chat manages chat operations.
type Chat struct {
	log   *logger.Logger
	users map[uuid.UUID]User
	mu    sync.RWMutex
}

// NewChat creates a new chat manager.
func NewChat(log *logger.Logger) *Chat {
	c := &Chat{
		log:   log,
		users: make(map[uuid.UUID]User),
	}

	c.ping()

	return c
}

// Handshake performs the handshake process for a new connection.
func (c *Chat) Handshake(ctx context.Context, w http.ResponseWriter, r *http.Request) (User, error) {
	var ws websocket.Upgrader
	conn, err := ws.Upgrade(w, r, nil)
	if err != nil {
		return User{}, errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket: %s", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte("HELLO"))
	if err != nil {
		return User{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	msg, err := c.readMessage(ctx, conn)
	if err != nil {
		return User{}, fmt.Errorf("read message: %w", err)
	}

	usr := User{
		Conn: conn,
	}
	if err := json.Unmarshal(msg, &usr); err != nil {
		return User{}, fmt.Errorf("unmarshal message: %w", err)
	}

	if err := c.addUser(ctx, usr); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("already connected")); err != nil {
			return User{}, fmt.Errorf("write message: %w", err)
		}

		return User{}, fmt.Errorf("add user: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("WELCOME "+usr.Name)); err != nil {
		return User{}, err
	}

	c.log.Info(ctx, "handshake complete", "user", usr)

	return usr, nil
}

// Listen starts listening for messages from the connected user.
func (c *Chat) Listen(ctx context.Context, usr User) {
	for {
		msg, err := c.readMessage(ctx, usr.Conn)
		if err != nil {
			c.log.Error(ctx, "read message failed", "error", err)
			return
		}

		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Error(ctx, "unmarshal message failed", "error", err)
			return
		}

		if err := c.sendMessage(ctx, inMsg); err != nil {
			c.log.Error(ctx, "send message failed", "error", err)
			continue
		}
	}
}

// ----------------------------------------------------------------------------------

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
		if resp.err != nil {
			return nil, resp.err
		}
	}

	return resp.msg, nil
}

// sendMessage sends a message from one user to another.
func (c *Chat) sendMessage(ctx context.Context, msg inMessage) error {
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
		From: User{
			ID:   from.ID,
			Name: from.Name,
		},
		To: User{
			ID:   to.ID,
			Name: to.Name,
		},
		Message: msg.Message,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		c.removeUser(ctx, to.ID)
		return fmt.Errorf("write json: %w", err)
	}

	return nil
}

// connections returns a copy of all current connections.
func (c *Chat) connections() map[uuid.UUID]*websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conns := make(map[uuid.UUID]*websocket.Conn, len(c.users))
	for id, usr := range c.users {
		conns[id] = usr.Conn
	}

	return conns
}

// ping sends periodic ping messages to all connected users.
func (c *Chat) ping() {
	ticket := time.NewTicker(time.Second * 10)

	go func() {
		for {
			ctx := context.Background()
			<-ticket.C

			conns := c.connections() // copy connections to avoid read locking during ping
			for k, conn := range conns {
				c.log.Info(ctx, "ping", "id", "k")
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.removeUser(context.Background(), k) // remove user on ping failure only so that there's a single source of truth
				}
			}

		}
	}()
}

// addUser adds a user and their connection to the chat.
// Returns an error if the user already exists.
func (c *Chat) addUser(ctx context.Context, usr User) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.users[usr.ID]; ok {
		return fmt.Errorf("user already exists")
	}

	c.users[usr.ID] = User{
		Conn: usr.Conn,
		ID:   usr.ID,
		Name: usr.Name,
	}

	c.log.Info(ctx, "added user", "user", usr.Name, "id", usr.ID)

	return nil
}

// removeUser removes a user from the chat.
func (c *Chat) removeUser(ctx context.Context, userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	usr, ok := c.users[userID]
	if !ok {
		c.log.Info(ctx, "remove user: user does not exist", "id", userID)
		return
	}

	c.log.Info(ctx, "removing user", "user", usr.Name, "id", usr.ID)

	delete(c.users, userID)
	usr.Conn.Close()
}
