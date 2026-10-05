package hub

import (
	"errors"
	"fmt"
	"testing"
)

func TestHubSubscriptionReferenceCounting(t *testing.T) {
	h := NewHub()
	c1 := NewClient(1, nil)
	c2 := NewClient(2, nil)
	h.Register(c1)
	h.Register(c2)
	cmd := ClientCommand{Type: "subscribe", Channel: "market", Symbol: "600176"}

	if err := h.Subscribe(c1, cmd); err != nil {
		t.Fatal(err)
	}
	if err := h.Subscribe(c1, cmd); err != nil {
		t.Fatal(err)
	}
	if err := h.Subscribe(c2, cmd); err != nil {
		t.Fatal(err)
	}
	if got := h.refs[SubscriptionKey("market", "600176", 0)]; got != 2 {
		t.Fatalf("expected two distinct subscribers, got %d", got)
	}

	h.Unregister(c1)
	if got := h.ActiveSymbols("market"); len(got) != 1 || got[0] != "600176" {
		t.Fatalf("symbol should remain active, got %#v", got)
	}
	h.Unregister(c2)
	if got := h.ActiveSymbols("market"); len(got) != 0 {
		t.Fatalf("symbol should be released, got %#v", got)
	}
}

func TestHubEnforcesMarketSubscriptionLimit(t *testing.T) {
	h := NewHub()
	c := NewClient(1, nil)
	h.Register(c)
	for i := 0; i < maxMarketSubscriptions; i++ {
		cmd := ClientCommand{Type: "subscribe", Channel: "market", Symbol: fmt.Sprintf("%06d", i)}
		if err := h.Subscribe(c, cmd); err != nil {
			t.Fatalf("subscription %d failed: %v", i, err)
		}
	}
	err := h.Subscribe(c, ClientCommand{Type: "subscribe", Channel: "market", Symbol: "999999"})
	if !errors.Is(err, ErrSubscriptionLimit) {
		t.Fatalf("expected subscription limit, got %v", err)
	}
	h.Unregister(c)
}
