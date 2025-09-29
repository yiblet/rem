# History Management Features Implementation Plan

## Current Project Status
**Feature**: History Management Enhancements
**Phase**: Planning and Design
**Goal**: Implement comprehensive history management features including configurable limits, environment variables, location customization, and advanced operations.

## History-Related Features from TODO.md

### Core History Configuration
- [ ] Make the history length limit configurable
  - [ ] Allow via TUI to change the configs
  - [ ] Allow via CLI to change the configs
- [ ] Add a clear command to clear rem history
- [ ] Add a REM_HISTORY environment variable for custom location configuration
- [ ] Change history location from `~/.config/rem/content/` to `~/.config/rem/history/`
- [ ] Allow rem to configure its history via a CLI `--history` command

### Advanced History Operations
- [ ] Allow via TUI the ability to delete individual items from history
- [ ] Add CLI-based search functionality - finds the first content in history matching a regex
  - Example: `rem search 'hello.*world'` returns the first file in history with a line matching the pattern

### Quality of Life Improvements
- [ ] Detect and don't show binary files in the preview
- [ ] Allow pager-style commands on the left sidebar in the TUI

## Feature Priority and Implementation Order

### Phase 1: History Location and Environment Configuration [COMPLETED]
**Priority**: High - Foundation for all other features
**Complexity**: Low-Medium
**Dependencies**: None

#### 1.1 REM_HISTORY Environment Variable
- Add support for `REM_HISTORY` environment variable
- Default to `~/.config/rem/history/` if not set
- Update `remfs` package to use configurable base path
- Maintain backward compatibility with existing `~/.config/rem/content/` locations

#### 1.2 Directory Migration
- Implement migration logic from `content/` to `history/`
- Add migration command or automatic detection
- Preserve existing user data during transition

#### 1.3 CLI History Flag
- Add `--history` flag to CLI commands
- Override environment variable when specified
- Update all CLI commands to respect history location setting

**Phase 1 Testing:**
- **Unit Tests**: Environment variable parsing and precedence logic in `remfs` package ✓
- **Unit Tests**: Directory migration functions with mock filesystem scenarios ✓
- **Integration Tests**: CLI commands with `--history` flag using temporary directories ✓
- **Integration Tests**: Backward compatibility with existing `content/` directory structure ✓
- **Unit Tests**: Error handling for invalid paths, missing permissions, and filesystem failures ✓

**Phase 1 Implementation Summary:**
- ✅ **REM_HISTORY Environment Variable**: Added support via go-arg's `env:` tag in CLI args
- ✅ **CLI --history Flag**: Added global flag that overrides environment variable and defaults
- ✅ **Directory Migration**: Automatic migration from `~/.config/rem/content/` to `~/.config/rem/history/`
- ✅ **NewWithHistoryPath Function**: Enhanced remfs package to support custom history paths
- ✅ **Precedence Logic**: CLI flag > REM_HISTORY env var > default `~/.config/rem/history/`
- ✅ **Backward Compatibility**: Safe migration with conflict detection and marker files
- ✅ **Test Coverage**: Comprehensive unit and integration tests for all new functionality

### Phase 2: Configuration System and CLI Management [COMPLETED]
**Priority**: High - Core functionality improvement
**Complexity**: Medium
**Dependencies**: Phase 1 complete

#### 2.1 Configuration System
- Create configuration file structure (`~/.config/rem/config.yaml` or similar)
- Define configuration schema with history limit setting
- Default to current 20-item limit for backward compatibility

#### 2.2 CLI Configuration
- Add `rem config` command with subcommands:
  - `rem config get history-limit`
  - `rem config set history-limit <number>`
  - `rem config list` (show all settings)

**Phase 2 Testing:**
- **Unit Tests**: Configuration file parsing, validation, and YAML marshaling/unmarshaling ✓
- **Integration Tests**: CLI config commands (`get`, `set`, `list`) with temporary config files ✓
- **Integration Tests**: History limit enforcement and auto-cleanup with mock stack data ✓
- **Unit Tests**: Configuration persistence and file I/O operations ✓

**Phase 2 Implementation Summary:**
- ✅ **Configuration System**: Created `internal/config` package with YAML-based config management
- ✅ **CLI Commands**: Added `rem config get/set/list` subcommands with validation
- ✅ **Stack Manager Integration**: Modified StackManager to use configurable history limits
- ✅ **Backward Compatibility**: Maintained default behavior with `DefaultMaxStackSize = 20`
- ✅ **Comprehensive Testing**: Unit tests for config package and integration tests for CLI commands
- ✅ **Configuration Schema**: Supports `history-limit`, `show-binary`, and `history-location` settings

### Phase 3: History Management Operations [TODO]
**Priority**: Medium - User convenience features
**Complexity**: Medium-High
**Dependencies**: Phase 2 complete

#### 3.1 Clear Command
- Add `rem clear` CLI command
- Confirmation prompt for safety
- Option for `--force` to skip confirmation
- Preserve configuration files

#### 3.2 Individual Item Deletion (TUI)
- Add keybinding for deletion (e.g., 'd' or 'Delete')
- Confirmation dialog for safety
- Update stack indices after deletion
- Refresh display immediately

#### 3.3 Search Functionality
- Add `rem search <pattern>` CLI command
- Support for regex patterns
- Return first matching item's content
- Option to return item index instead of content
- Integration with existing stream-based content model

**Phase 3 Testing:**
- **Integration Tests**: `rem clear` command with mock filesystem and confirmation logic
- **Unit Tests**: TUI deletion component state management and stack reordering logic
- **Unit Tests**: Search regex compilation, pattern matching, and result formatting
- **Integration Tests**: Search functionality with generated test data and various file sizes
- **Unit Tests**: Error handling for malformed regex patterns and empty result sets

### Phase 4: TUI Configuration Interface [TODO]
**Priority**: High - User experience improvement
**Complexity**: Medium
**Dependencies**: Phase 2 and 3 complete

#### 4.1 TUI Configuration Screen
- Add configuration screen accessible via keybinding (e.g., 'c')
- Allow real-time adjustment of history limit
- Show current usage vs. limit
- Visual feedback for configuration changes

**Phase 4 Testing:**
- **Unit Tests**: TUI configuration component model updates and view rendering
- **Unit Tests**: Configuration screen key handling and state management
- **Integration Tests**: Configuration changes persist to file and affect stack behavior
- **Unit Tests**: Real-time validation and error display for invalid values

### Phase 5: Quality of Life Enhancements [TODO]
**Priority**: Low-Medium - Polish and user experience
**Complexity**: Low-Medium
**Dependencies**: Core features complete

#### 5.1 Binary File Detection
- Implement binary file detection in preview system
- Add fallback display for binary files (filename, size, type)
- Configurable binary detection sensitivity

#### 5.2 Enhanced TUI Navigation
- Add pager-style commands to left pane:
  - 'f'/'F' - page forward/backward
  - '/'/'?' - search within item list
  - 'gg'/'G' - go to top/bottom
- Consistent keybinding scheme across panes

**Phase 5 Testing:**
- **Unit Tests**: Binary file detection algorithm with generated binary/text test data
- **Unit Tests**: Binary file metadata extraction and fallback display formatting
- **Unit Tests**: TUI navigation component key handling and state transitions
- **Integration Tests**: Pager-style commands with mock TUI models and view rendering
- **Integration Tests**: Performance benchmarks with mixed content types using generated data

## Technical Implementation Details

### Configuration Architecture
```go
type Config struct {
    HistoryLimit    int    `yaml:"history_limit"`
    HistoryLocation string `yaml:"history_location,omitempty"`
    ShowBinary      bool   `yaml:"show_binary"`
}
```

### Environment Variable Precedence
1. CLI `--history` flag (highest)
2. `REM_HISTORY` environment variable
3. Configuration file setting
4. Default `~/.config/rem/history/` (lowest)

### Directory Structure Changes
```
~/.config/rem/
├── history/                 # New default (was content/)
│   ├── 2025-09-28T10-15-30.123456-07-00.txt
│   └── ...
├── config.yaml             # New configuration file
└── .migration_complete     # Migration marker
```

### Search Implementation
- Leverage existing `StackItem.Search()` method
- Implement regex compilation and caching
- Stream-based searching for memory efficiency
- Return structured results with match context

## Testing Strategy

Each phase must include high-level test descriptions to ensure correctness of the implementation. Tests should be specified at the phase level and implemented alongside the features.

## Success Criteria

### Functional Requirements
- [ ] Users can configure history limit via CLI and TUI
- [ ] REM_HISTORY environment variable works correctly
- [ ] Migration from old location is seamless
- [ ] Clear command safely removes all history
- [ ] Individual item deletion works in TUI
- [ ] Search command finds and returns correct matches
- [ ] Binary files are handled appropriately

### Compatibility Requirements
- [ ] Backward compatibility with existing installations
- [ ] No breaking changes to existing CLI interface
- [ ] Existing TUI keybindings remain unchanged
- [ ] Configuration is optional (sensible defaults)

### Quality Requirements
- [ ] All new functionality has comprehensive tests
- [ ] Error handling for invalid configurations
- [ ] Clear user feedback for all operations
- [ ] Performance remains acceptable with larger history limits

## Risk Mitigation

### Data Safety
- Always backup existing data during migration
- Implement transaction-like operations for critical changes
- Extensive testing with various data scenarios

### Backward Compatibility
- Support both old and new directory structures during transition
- Graceful fallback for missing configuration
- Clear migration path documentation

### Performance
- Efficient search algorithms for large history
- Lazy loading for TUI with many items
- Memory-conscious operations for large files

## Next Steps

1. **Start Phase 1**: Begin with environment variable support and directory migration
2. **Create Feature Branch**: `feat/history-management`
3. **Incremental Implementation**: Each phase should be fully functional
4. **Continuous Testing**: Maintain test coverage throughout
5. **Documentation**: Update ARCHITECTURE.md with new configuration system

---

**Last Updated**: 2025-09-29
**Status**: Phase 2 Completed - Ready for Phase 3