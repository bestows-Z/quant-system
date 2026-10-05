package hub

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

var ErrSubscriptionLimit = errors.New("subscription limit exceeded")

const (
	maxMarketSubscriptions = 30
	maxSectorSubscriptions = 10
)

type Hub struct {
	mu      sync.RWMutex
	clients map[uint64]map[*Client]struct{}
	refs    map[string]int
}

func NewHub() *Hub {
	return &Hub{clients: make(map[uint64]map[*Client]struct{}), refs: make(map[string]int)}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = make(map[*Client]struct{})
	}
	h.clients[c.UserID][c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if items := h.clients[c.UserID]; items != nil {
		delete(items, c)
		if len(items) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	for _, key := range c.Subscriptions() {
		h.decrementRef(key)
	}
	h.mu.Unlock()
	c.Close()
}

func (h *Hub) Subscribe(c *Client, cmd ClientCommand) error {
	key := SubscriptionKey(cmd.Channel, cmd.Symbol, cmd.AccountID)
	if c.HasSubscription(key) {
		return nil
	}
	limit := maxMarketSubscriptions
	if cmd.Channel == "sector" {
		limit = maxSectorSubscriptions
	}
	if c.SubscriptionCount(cmd.Channel) >= limit {
		return ErrSubscriptionLimit
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if c.Subscribe(key) {
		h.refs[key]++
	}
	return nil
}

func (h *Hub) Unsubscribe(c *Client, cmd ClientCommand) {
	key := SubscriptionKey(cmd.Channel, cmd.Symbol, cmd.AccountID)
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.Unsubscribe(key) {
		h.decrementRef(key)
	}
}

func (h *Hub) decrementRef(key string) {
	if h.refs[key] <= 1 {
		delete(h.refs, key)
		return
	}
	h.refs[key]--
}

func (h *Hub) ActiveSymbols(channel string) []string {
	prefix := channel + ":symbol:"
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := make([]string, 0)
	for key, refs := range h.refs {
		if refs > 0 && strings.HasPrefix(key, prefix) {
			items = append(items, strings.TrimPrefix(key, prefix))
		}
	}
	sort.Strings(items)
	return items
}

func (h *Hub) PushUser(userID uint64, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		c.Send(msg)
	}
}

func (h *Hub) BroadcastSubscribed(channel, symbol string, accountID uint64, msg Message) {
	key := SubscriptionKey(channel, symbol, accountID)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, clients := range h.clients {
		for c := range clients {
			if c.HasSubscription(key) || c.HasSubscription(channel) {
				c.Send(msg)
			}
		}
	}
}
