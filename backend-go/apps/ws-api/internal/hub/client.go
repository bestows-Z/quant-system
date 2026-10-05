package hub

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 70 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 4096
)

type Client struct {
	UserID uint64
	Conn   *websocket.Conn
	send   chan Message
	mu     sync.RWMutex
	subs   map[string]struct{}
	close  sync.Once
}

func NewClient(userID uint64, conn *websocket.Conn) *Client {
	return &Client{UserID: userID, Conn: conn, send: make(chan Message, 32), subs: make(map[string]struct{})}
}

func (c *Client) Send(msg Message) {
	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) Close() {
	c.close.Do(func() {
		close(c.send)
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
	})
}

func (c *Client) HasSubscription(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subs[key]
	return ok
}

func (c *Client) Subscribe(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.subs[key]; exists {
		return false
	}
	c.subs[key] = struct{}{}
	return true
}

func (c *Client) Unsubscribe(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.subs[key]; !exists {
		return false
	}
	delete(c.subs, key)
	return true
}

func (c *Client) Subscriptions() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := make([]string, 0, len(c.subs))
	for key := range c.subs {
		items = append(items, key)
	}
	return items
}

func (c *Client) SubscriptionCount(channel string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	prefix := channel + ":"
	for key := range c.subs {
		if key == channel || len(key) > len(prefix) && key[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

func (c *Client) ReadPump(onCommand func(ClientCommand)) {
	defer c.Close()
	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, body, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		var cmd ClientCommand
		if err := json.Unmarshal(body, &cmd); err != nil {
			continue
		}
		onCommand(cmd)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.Conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
