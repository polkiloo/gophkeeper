// Package buildinfo prints build metadata.
package buildinfo

import (
	"fmt"
	"io"
)

// Info carries build metadata.
type Info struct {
	Version string
	Date    string
	Commit  string
}

// Print writes build metadata to the writer.
func Print(w io.Writer, info Info) {
	fmt.Fprintf(w, "version=%s date=%s commit=%s\n", info.Version, info.Date, info.Commit)
}
