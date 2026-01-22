package outbound

import "time"

// Clock provides current time.
type Clock interface {
	Now() time.Time
}

// IDGenerator provides stable identifiers.
type IDGenerator interface {
	NewID() string
}
