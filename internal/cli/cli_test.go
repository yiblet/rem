package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWithArgs_DefaultDB(t *testing.T) {
	// Create temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test CLI creation without custom database path
	args := &Args{}
	cli, err := NewWithArgs(args)
	if err != nil {
		t.Fatalf("NewWithArgs failed: %v", err)
	}

	// Verify database was created in default location
	expectedDBPath := filepath.Join(tempDir, ".config", "rem", "rem.db")
	if _, err := os.Stat(expectedDBPath); os.IsNotExist(err) {
		t.Errorf("Expected database at %s, but it doesn't exist", expectedDBPath)
	}

	// Cleanup
	if cli.store != nil {
		cli.store.Close()
	}
}

func TestNewWithArgs_CustomDBPath(t *testing.T) {
	// Create temporary directory for custom database
	tempDir := t.TempDir()
	customDBPath := filepath.Join(tempDir, "custom.db")

	// Test CLI creation with custom database path
	args := &Args{
		DBPath: &customDBPath,
	}

	cli, err := NewWithArgs(args)
	if err != nil {
		t.Fatalf("NewWithArgs with custom path failed: %v", err)
	}

	// Check that database was created at custom location
	if _, err := os.Stat(customDBPath); os.IsNotExist(err) {
		t.Errorf("Custom database should be created: %s", customDBPath)
	}

	// Cleanup
	if cli.store != nil {
		cli.store.Close()
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

	// Verify database was created in default location
	expectedDBPath := filepath.Join(tempDir, ".config", "rem", "rem.db")
	if _, err := os.Stat(expectedDBPath); os.IsNotExist(err) {
		t.Errorf("Expected database at %s, but it doesn't exist", expectedDBPath)
	}

	// Cleanup
	if cli.store != nil {
		cli.store.Close()
	}
}

func TestNew_DefaultBehavior(t *testing.T) {
	// Create temporary home directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test CLI creation using New() function
	cli, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Verify database was created in default location
	expectedDBPath := filepath.Join(tempDir, ".config", "rem", "rem.db")
	if _, err := os.Stat(expectedDBPath); os.IsNotExist(err) {
		t.Errorf("Expected database at %s, but it doesn't exist", expectedDBPath)
	}

	// Cleanup
	if cli.store != nil {
		cli.store.Close()
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
			name: "with custom db path",
			args: Args{
				DBPath: stringPtr("/tmp/custom.db"),
				Get:    &GetCmd{},
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
					Get: &ConfigGetCmd{Key: "history_limit"},
				},
			},
			expectErr: false,
		},
		{
			name: "config set valid",
			args: Args{
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: "history_limit", Value: "100"},
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
					Get:  &ConfigGetCmd{Key: "history_limit"},
					Set:  &ConfigSetCmd{Key: "history_limit", Value: "100"},
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
		defer cli.store.Close()

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
				Get: &ConfigGetCmd{Key: "history_limit"},
			},
		}

		cli, err := NewWithArgs(args)
		if err != nil {
			t.Fatalf("Failed to create CLI: %v", err)
		}
		defer cli.store.Close()

		err = cli.Execute(args)
		if err != nil {
			t.Errorf("config get failed: %v", err)
		}
	})

	// Test config set and get cycle
	t.Run("config set and get cycle", func(t *testing.T) {
		// Use custom DB path for this test
		dbPath := filepath.Join(tempDir, "test-config.db")

		// First, set a value
		setArgs := &Args{
			DBPath: &dbPath,
			Config: &ConfigCmd{
				Set: &ConfigSetCmd{Key: "history_limit", Value: "50"},
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
		cli.store.Close()

		// Then, get the value to verify
		getArgs := &Args{
			DBPath: &dbPath,
			Config: &ConfigCmd{
				Get: &ConfigGetCmd{Key: "history_limit"},
			},
		}

		cli2, err := NewWithArgs(getArgs)
		if err != nil {
			t.Fatalf("Failed to create CLI for get: %v", err)
		}
		defer cli2.store.Close()

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
			{"history_limit", "75"},
			{"show_binary", "true"},
		}

		for _, tc := range testCases {
			dbPath := filepath.Join(tempDir, "test-config-"+tc.key+".db")
			args := &Args{
				DBPath: &dbPath,
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: tc.key, Value: tc.value},
				},
			}

			cli, err := NewWithArgs(args)
			if err != nil {
				t.Fatalf("Failed to create CLI for %s: %v", tc.key, err)
			}

			err = cli.Execute(args)
			cli.store.Close()
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
			{"history_limit", "not-a-number"},
			{"history_limit", "-5"},
			{"show_binary", "maybe"},
		}

		for _, tc := range testCases {
			dbPath := filepath.Join(tempDir, "test-invalid-"+tc.key+".db")
			args := &Args{
				DBPath: &dbPath,
				Config: &ConfigCmd{
					Set: &ConfigSetCmd{Key: tc.key, Value: tc.value},
				},
			}

			cli, err := NewWithArgs(args)
			if err != nil {
				t.Fatalf("Failed to create CLI for %s: %v", tc.key, err)
			}

			err = cli.Execute(args)
			cli.store.Close()
			if err == nil {
				t.Errorf("Expected config set %s=%s to fail, but it succeeded", tc.key, tc.value)
			}
		}
	})
}

func TestConfigIntegrationWithQueueManager(t *testing.T) {
	// Create temporary directories for test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Set a custom history limit
	setArgs := &Args{
		DBPath: &dbPath,
		Config: &ConfigCmd{
			Set: &ConfigSetCmd{Key: "history_limit", Value: "5"},
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
	cli.store.Close()

	// Create a new CLI instance that should pick up the configuration
	cli2, err := NewWithArgs(&Args{DBPath: &dbPath})
	if err != nil {
		t.Fatalf("Failed to create CLI after config: %v", err)
	}
	defer cli2.store.Close()

	// Verify the queue manager has the correct history limit
	if cli2.queueManager.GetHistoryLimit() != 5 {
		t.Errorf("Expected queue manager history limit 5, got %d", cli2.queueManager.GetHistoryLimit())
	}
}

func TestSearchCommand_BasicTitleSearch(t *testing.T) {
	// Create CLI with custom database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "search-test.db")
	args := &Args{DBPath: &dbPath}
	cli, err := NewWithArgs(args)
	if err != nil {
		t.Fatalf("Failed to create CLI: %v", err)
	}
	defer cli.store.Close()

	// Add test data with titles
	testData := []struct {
		content string
		title   string
	}{
		{"first item content", "pattern match"},
		{"second item content", "no keyword"},
		{"third item content", "another pattern"},
	}

	for _, data := range testData {
		_, err := cli.queueManager.Enqueue(strings.NewReader(data.content), data.title)
		if err != nil {
			t.Fatalf("Failed to enqueue test data: %v", err)
		}
	}

	// Test basic title search
	t.Run("title search finds matches", func(t *testing.T) {
		searchCmd := &SearchCmd{
			Pattern: "pattern",
		}

		// Execute search - this would write to stdout in real usage
		// For testing, we verify no error occurs
		err := cli.executeSearch(searchCmd)
		if err != nil {
			t.Errorf("Search failed: %v", err)
		}
	})

	// Test search with no matches
	t.Run("search with no matches returns error", func(t *testing.T) {
		searchCmd := &SearchCmd{
			Pattern: "nonexistent",
		}

		err := cli.executeSearch(searchCmd)
		if err == nil {
			t.Errorf("Expected search to fail with no matches")
		}
	})
}

// Helper functions for pointer creation
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
