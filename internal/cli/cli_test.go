package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yiblet/rem/internal/remfs"
)

func TestNewWithArgs_DefaultHistory(t *testing.T) {
	// Create temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test CLI creation without custom history path
	args := &Args{}
	cli, err := NewWithArgs(args)
	if err != nil {
		t.Fatalf("NewWithArgs failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, remfs.ConfigDir, remfs.DefaultHistDir)
	if cli.filesystem.(*remfs.RemFS).Root() != expectedPath {
		t.Errorf("Expected filesystem root %s, got %s",
			expectedPath, cli.filesystem.(*remfs.RemFS).Root())
	}
}

func TestNewWithArgs_CustomHistoryPath(t *testing.T) {
	// Create temporary directory for custom history
	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "my-custom-history")

	// Test CLI creation with custom history path
	historyPath := customPath
	args := &Args{
		History: &historyPath,
	}

	cli, err := NewWithArgs(args)
	if err != nil {
		t.Fatalf("NewWithArgs with custom path failed: %v", err)
	}

	if cli.filesystem.(*remfs.RemFS).Root() != customPath {
		t.Errorf("Expected filesystem root %s, got %s",
			customPath, cli.filesystem.(*remfs.RemFS).Root())
	}

	// Check that directory was created
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Errorf("Custom history directory should be created: %s", customPath)
	}
}

func TestNewWithArgs_RelativeHistoryPath(t *testing.T) {
	// Create temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test CLI creation with relative history path
	relativePath := "custom-rel-path"
	args := &Args{
		History: &relativePath,
	}

	cli, err := NewWithArgs(args)
	if err != nil {
		t.Fatalf("NewWithArgs with relative path failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, remfs.ConfigDir, relativePath)
	if cli.filesystem.(*remfs.RemFS).Root() != expectedPath {
		t.Errorf("Expected filesystem root %s, got %s",
			expectedPath, cli.filesystem.(*remfs.RemFS).Root())
	}
}

func TestNewWithArgs_NilArgs(t *testing.T) {
	// Create temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test CLI creation with nil args (should use defaults)
	cli, err := NewWithArgs(nil)
	if err != nil {
		t.Fatalf("NewWithArgs with nil args failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, remfs.ConfigDir, remfs.DefaultHistDir)
	if cli.filesystem.(*remfs.RemFS).Root() != expectedPath {
		t.Errorf("Expected filesystem root %s, got %s",
			expectedPath, cli.filesystem.(*remfs.RemFS).Root())
	}
}

func TestNew_DefaultBehavior(t *testing.T) {
	// Create temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test CLI creation using legacy New() function
	cli, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, remfs.ConfigDir, remfs.DefaultHistDir)
	if cli.filesystem.(*remfs.RemFS).Root() != expectedPath {
		t.Errorf("Expected filesystem root %s, got %s",
			expectedPath, cli.filesystem.(*remfs.RemFS).Root())
	}
}

func TestArgsValidation_ValidCases(t *testing.T) {
	tests := []struct {
		name string
		args Args
	}{
		{
			name: "store from file",
			args: Args{
				Store: &StoreCmd{
					Files: []string{"test.txt"},
				},
			},
		},
		{
			name: "store from clipboard",
			args: Args{
				Store: &StoreCmd{
					Clipboard: true,
				},
			},
		},
		{
			name: "store from stdin",
			args: Args{
				Store: &StoreCmd{},
			},
		},
		{
			name: "get with index",
			args: Args{
				Get: &GetCmd{
					Index: intPtr(0),
				},
			},
		},
		{
			name: "get to file",
			args: Args{
				Get: &GetCmd{
					Index: intPtr(1),
					File:  stringPtr("output.txt"),
				},
			},
		},
		{
			name: "get to clipboard",
			args: Args{
				Get: &GetCmd{
					Index:     intPtr(2),
					Clipboard: true,
				},
			},
		},
		{
			name: "with custom history",
			args: Args{
				History: stringPtr("/tmp/custom-history"),
				Get:     &GetCmd{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Validate()
			if err != nil {
				t.Errorf("Expected validation to pass for %s, got: %v", tt.name, err)
			}
		})
	}
}

func TestArgsValidation_InvalidCases(t *testing.T) {
	tests := []struct {
		name string
		args Args
	}{
		{
			name: "store both file and clipboard",
			args: Args{
				Store: &StoreCmd{
					Files:     []string{"test.txt"},
					Clipboard: true,
				},
			},
		},
		{
			name: "get both file and clipboard",
			args: Args{
				Get: &GetCmd{
					Index:     intPtr(0),
					File:      stringPtr("output.txt"),
					Clipboard: true,
				},
			},
		},
		{
			name: "get negative index",
			args: Args{
				Get: &GetCmd{
					Index: intPtr(-1),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Validate()
			if err == nil {
				t.Errorf("Expected validation to fail for %s", tt.name)
			}
		})
	}
}

func TestConfigCommands_ValidationCases(t *testing.T) {
	tests := []struct {
		name      string
		args      Args
		expectErr bool
	}{
		{
			name: "config get valid",
			args: Args{
				Config: &ConfigCmd{
					Get: &ConfigGetCmd{Key: "history-limit"},
				},
			},
			expectErr: false,
		},
		{
			name: "config set valid",
			args: Args{
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: "history-limit", Value: "100"},
				},
			},
			expectErr: false,
		},
		{
			name: "config list valid",
			args: Args{
				Config: &ConfigCmd{
					List: &ConfigListCmd{},
				},
			},
			expectErr: false,
		},
		{
			name: "config get invalid key",
			args: Args{
				Config: &ConfigCmd{
					Get: &ConfigGetCmd{Key: "invalid-key"},
				},
			},
			expectErr: true,
		},
		{
			name: "config set invalid key",
			args: Args{
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: "invalid-key", Value: "value"},
				},
			},
			expectErr: true,
		},
		{
			name: "config no subcommand",
			args: Args{
				Config: &ConfigCmd{},
			},
			expectErr: true,
		},
		{
			name: "config multiple subcommands",
			args: Args{
				Config: &ConfigCmd{
					Get:  &ConfigGetCmd{Key: "history-limit"},
					Set:  &ConfigSetCmd{Key: "history-limit", Value: "100"},
					List: &ConfigListCmd{},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Validate()
			if tt.expectErr && err == nil {
				t.Errorf("Expected validation to fail for %s", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected validation to pass for %s, got: %v", tt.name, err)
			}
		})
	}
}

func TestConfigCommands_Integration(t *testing.T) {
	// Create temporary directories for test
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test config list with default values
	t.Run("config list default", func(t *testing.T) {
		args := &Args{
			Config: &ConfigCmd{
				List: &ConfigListCmd{},
			},
		}

		cli, err := NewWithArgs(args)
		if err != nil {
			t.Fatalf("Failed to create CLI: %v", err)
		}

		// Capture output would require more complex setup,
		// but we can test that it doesn't error
		err = cli.Execute(args)
		if err != nil {
			t.Errorf("config list failed: %v", err)
		}
	})

	// Test config get
	t.Run("config get default value", func(t *testing.T) {
		args := &Args{
			Config: &ConfigCmd{
				Get: &ConfigGetCmd{Key: "history-limit"},
			},
		}

		cli, err := NewWithArgs(args)
		if err != nil {
			t.Fatalf("Failed to create CLI: %v", err)
		}

		err = cli.Execute(args)
		if err != nil {
			t.Errorf("config get failed: %v", err)
		}
	})

	// Test config set and get cycle
	t.Run("config set and get cycle", func(t *testing.T) {
		// First, set a value
		setArgs := &Args{
			Config: &ConfigCmd{
				Set: &ConfigSetCmd{Key: "history-limit", Value: "50"},
			},
		}

		cli, err := NewWithArgs(setArgs)
		if err != nil {
			t.Fatalf("Failed to create CLI for set: %v", err)
		}

		err = cli.Execute(setArgs)
		if err != nil {
			t.Errorf("config set failed: %v", err)
		}

		// Then, get the value to verify
		getArgs := &Args{
			Config: &ConfigCmd{
				Get: &ConfigGetCmd{Key: "history-limit"},
			},
		}

		cli2, err := NewWithArgs(getArgs)
		if err != nil {
			t.Fatalf("Failed to create CLI for get: %v", err)
		}

		err = cli2.Execute(getArgs)
		if err != nil {
			t.Errorf("config get failed: %v", err)
		}
	})

	// Test config set with various types
	t.Run("config set different types", func(t *testing.T) {
		testCases := []struct {
			key   string
			value string
		}{
			{"history-limit", "75"},
			{"show-binary", "true"},
			{"history-location", "/custom/test/path"},
		}

		for _, tc := range testCases {
			args := &Args{
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: tc.key, Value: tc.value},
				},
			}

			cli, err := NewWithArgs(args)
			if err != nil {
				t.Fatalf("Failed to create CLI for %s: %v", tc.key, err)
			}

			err = cli.Execute(args)
			if err != nil {
				t.Errorf("config set %s=%s failed: %v", tc.key, tc.value, err)
			}
		}
	})

	// Test config set with invalid values
	t.Run("config set invalid values", func(t *testing.T) {
		testCases := []struct {
			key   string
			value string
		}{
			{"history-limit", "not-a-number"},
			{"history-limit", "-5"},
			{"history-limit", "2000"},
			{"show-binary", "maybe"},
		}

		for _, tc := range testCases {
			args := &Args{
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: tc.key, Value: tc.value},
				},
			}

			cli, err := NewWithArgs(args)
			if err != nil {
				t.Fatalf("Failed to create CLI for %s: %v", tc.key, err)
			}

			err = cli.Execute(args)
			if err == nil {
				t.Errorf("Expected config set %s=%s to fail, but it succeeded", tc.key, tc.value)
			}
		}
	})
}

func TestConfigIntegrationWithQueueManager(t *testing.T) {
	// Create temporary directories for test
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Set a custom history limit
	setArgs := &Args{
		Config: &ConfigCmd{
			Set: &ConfigSetCmd{Key: "history-limit", Value: "5"},
		},
	}

	cli, err := NewWithArgs(setArgs)
	if err != nil {
		t.Fatalf("Failed to create CLI for config set: %v", err)
	}

	err = cli.Execute(setArgs)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a new CLI instance that should pick up the configuration
	cli2, err := NewWithArgs(nil)
	if err != nil {
		t.Fatalf("Failed to create CLI after config: %v", err)
	}

	// Verify the queue manager has the correct history limit
	if cli2.stackManager.GetHistoryLimit() != 5 {
		t.Errorf("Expected queue manager history limit 5, got %d", cli2.stackManager.GetHistoryLimit())
	}
}

// Helper functions for pointer creation
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
