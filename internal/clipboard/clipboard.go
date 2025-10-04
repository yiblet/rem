package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

// Read reads text content from the system clipboard
func Read() ([]byte, error) {
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

// Write writes text content to the system clipboard
func Write(data []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return writeMac(data)
	case "linux":
		return writeLinux(data)
	case "windows":
		return fmt.Errorf("clipboard operations not supported on Windows")
	default:
		return fmt.Errorf("clipboard operations not supported on %s", runtime.GOOS)
	}
}

// readMac reads from clipboard on macOS using pbpaste
func readMac() ([]byte, error) {
	cmd := exec.Command("pbpaste")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run pbpaste: %w", err)
	}

	return out.Bytes(), nil
}

// writeMac writes to clipboard on macOS using pbcopy
func writeMac(data []byte) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader(data)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run pbcopy: %w", err)
	}

	return nil
}

// readLinux reads from clipboard on Linux using xclip or xsel
func readLinux() ([]byte, error) {
	// Try xclip first
	if data, err := readWithCommand("xclip", "-selection", "clipboard", "-o"); err == nil {
		return data, nil
	}

	// Fall back to xsel
	data, err := readWithCommand("xsel", "--clipboard", "--output")
	if err != nil {
		return nil, fmt.Errorf("failed to read clipboard (tried xclip and xsel): %w", err)
	}

	return data, nil
}

// writeLinux writes to clipboard on Linux using xclip or xsel
func writeLinux(data []byte) error {
	// Try xclip first
	if err := writeWithCommand(data, "xclip", "-selection", "clipboard"); err == nil {
		return nil
	}

	// Fall back to xsel
	if err := writeWithCommand(data, "xsel", "--clipboard", "--input"); err != nil {
		return fmt.Errorf("failed to write clipboard (tried xclip and xsel): %w", err)
	}

	return nil
}

// readWithCommand executes a command and returns its output
func readWithCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// writeWithCommand executes a command with data as stdin
func writeWithCommand(data []byte, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(data)

	return cmd.Run()
}
