package configutil

import (
	"os"
	"strconv"
	"time"
)

// Parser converts a string into a typed value.
type Parser[T any] func(string) (T, error)

// OverrideEnv updates dst if the environment variable is set and parsed.
func OverrideEnv[T any](dst *T, key string, parse Parser[T]) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return false
	}
	parsed, err := parse(value)
	if err != nil {
		return false
	}
	*dst = parsed
	return true
}

// ParseString returns the provided string.
func ParseString(value string) (string, error) {
	return value, nil
}

// ParseBool parses a boolean.
func ParseBool(value string) (bool, error) {
	return strconv.ParseBool(value)
}

// ParseDuration parses a duration.
func ParseDuration(value string) (time.Duration, error) {
	return time.ParseDuration(value)
}
