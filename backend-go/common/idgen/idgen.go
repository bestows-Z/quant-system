package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

func NewEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "event-unavailable"
	}
	return hex.EncodeToString(b[:])
}
