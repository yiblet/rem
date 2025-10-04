// Package clipboard provides an abstraction for system clipboard operations.
// It supports multiple implementations including system clipboard (pbcopy/pbpaste on macOS,
// xclip/xsel on Linux) and mock clipboard for testing.
//
// To use the clipboard, create an instance using one of the implementation constructors:
//   - sysboard.New() for system clipboard
//   - mockboard.New() for testing
package clipboard

import "io"

// Clipboard defines the interface for clipboard operations
type Clipboard interface {
	Read() (io.ReadCloser, error)
	Write(r io.Reader) error
	// IsSupported returns true if clipboard operations are supported on this system
	IsSupported() bool
}
