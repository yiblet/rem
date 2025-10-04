// Package sysboard implements system clipboard operations using platform-specific commands.
// On macOS it uses pbcopy/pbpaste, on Linux it uses xclip or xsel as a fallback.
package sysboard

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

// SystemClipboard implements Clipboard using system commands
type SystemClipboard struct{}

// New creates a new SystemClipboard instance
func New() *SystemClipboard {
	return &SystemClipboard{}
}

// IsSupported returns true if clipboard operations are supported on this system
func (s *SystemClipboard) IsSupported() bool {
	switch runtime.GOOS {
	case "darwin":
		// Check if pbcopy/pbpaste are available
		if _, err := exec.LookPath("pbcopy"); err != nil {
			return false
		}
		if _, err := exec.LookPath("pbpaste"); err != nil {
			return false
		}
		return true
	case "linux":
		// Check if xclip or xsel are available
		if _, err := exec.LookPath("xclip"); err == nil {
			return true
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return true
		}
		return false
	case "windows":
		return false
	default:
		return false
	}
}

// Read implements Clipboard.Read for SystemClipboard
func (s *SystemClipboard) Read() (io.ReadCloser, error) {
	switch runtime.GOOS {
	case "darwin":
		return readMac()
	case "linux":
		return readLinux()
	case "windows":
		return nil, fmt.Errorf("clipboard operations not supported on Windows")
	default:
		return nil, fmt.Errorf("clipboard operations not supported on %s", runtime.GOOS)
	}
}

// Write implements Clipboard.Write for SystemClipboard
func (s *SystemClipboard) Write(r io.Reader) error {
	switch runtime.GOOS {
	case "darwin":
		return writeMac(r)
	case "linux":
		return writeLinux(r)
	case "windows":
		return fmt.Errorf("clipboard operations not supported on Windows")
	default:
		return fmt.Errorf("clipboard operations not supported on %s", runtime.GOOS)
	}
}

// cmdReadCloser wraps a command's stdout and ensures the command is waited on when closed
type cmdReadCloser struct {
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (c *cmdReadCloser) Read(p []byte) (n int, err error) {
	return c.stdout.Read(p)
}

func (c *cmdReadCloser) Close() error {
	// Close stdout first
	if err := c.stdout.Close(); err != nil {
		c.cmd.Wait() // Still wait for command even if close fails
		return err
	}

	if runtime.GOOS != "windows" {
		c.cmd.Process.Signal(os.Interrupt) // send interrupt signal to kill process
	}
	// Wait for command to finish
	return c.cmd.Wait()
}

// readMac reads from clipboard on macOS using pbpaste
func readMac() (io.ReadCloser, error) {
	cmd := exec.Command("pbpaste")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pbpaste: %w", err)
	}

	return &cmdReadCloser{stdout: stdout, cmd: cmd}, nil
}

// writeMac writes to clipboard on macOS using pbcopy
func writeMac(r io.Reader) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = r

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run pbcopy: %w", err)
	}

	return nil
}

// readLinux reads from clipboard on Linux using xclip or xsel
func readLinux() (io.ReadCloser, error) {
	// Try xclip first
	if reader, err := readWithCommand("xclip", "-selection", "clipboard", "-o"); err == nil {
		return reader, nil
	}

	// Fall back to xsel
	reader, err := readWithCommand("xsel", "--clipboard", "--output")
	if err != nil {
		return nil, fmt.Errorf("failed to read clipboard (tried xclip and xsel): %w", err)
	}

	return reader, nil
}

// writeLinux writes to clipboard on Linux using xclip or xsel
func writeLinux(r io.Reader) error {
	// Try xclip first
	if err := writeWithCommand(r, "xclip", "-selection", "clipboard"); err == nil {
		return nil
	}

	// Fall back to xsel
	if err := writeWithCommand(r, "xsel", "--clipboard", "--input"); err != nil {
		return fmt.Errorf("failed to write clipboard (tried xclip and xsel): %w", err)
	}

	return nil
}

// readWithCommand executes a command and returns its output as a stream
func readWithCommand(name string, args ...string) (io.ReadCloser, error) {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &cmdReadCloser{stdout: stdout, cmd: cmd}, nil
}

// writeWithCommand executes a command with data as stdin
func writeWithCommand(r io.Reader, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = r

	return cmd.Run()
}
