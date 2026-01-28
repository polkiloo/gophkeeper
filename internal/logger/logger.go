// Package logger provides logging facilities.
package logger

import (
	"log"
	"os"
)

// New constructs a logger writing to stdout.
func New() *log.Logger {
	return log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
}
