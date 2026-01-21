package system

import (
	"crypto/rand"
	"encoding/hex"
)

// IDGenerator produces random hex identifiers.
type IDGenerator struct{}

// NewID generates a new identifier.
func (IDGenerator) NewID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
