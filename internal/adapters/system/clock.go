// Package system provides production system adapters.
package system

import "time"

// Clock implements outbound.Clock using time.Now.
type Clock struct{}

// Now returns the current time.
func (Clock) Now() time.Time {
	return time.Now()
}
