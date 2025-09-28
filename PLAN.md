# rem Project Implementation Plan

## Project Status

**Current State**: ✅ **Phase 2 Complete** - Core Queue Manager & Storage
**Next Phase**: 🚧 **Phase 3** - Full CLI Interface (store/get commands)

## Implementation Phases

### ✅ Phase 1: Interactive TUI Foundation (COMPLETED)
**Goal**: Build a robust dual-pane TUI viewer with pager functionality

#### Completed Components:
- [x] **Dual-pane TUI layout** with responsive sizing
- [x] **Queue navigation** (left pane) with auto-selection
- [x] **Content pager** (right pane) with full less-like functionality
- [x] **Regex search** with highlighting and match navigation
- [x] **Per-item state tracking** (scroll position, search state)
- [x] **Stream-based content model** (`io.ReadSeekCloser`)
- [x] **Text wrapping** and proper truncation
- [x] **Status line** with search input and feedback
- [x] **Modular architecture** in `internal/tui/` package

#### Current Capabilities:
```bash
go run main.go    # Launch interactive viewer with 5 dummy items
# TUI Features: Tab (switch panes), j/k (navigate), /pattern (search), n/N (next/prev match)
```

---

### ✅ Phase 2: Core Queue Manager & Storage (COMPLETED)
**Goal**: Implement persistent queue with file-based storage

#### Completed Components:
- [x] **Config directory setup** (`~/.config/rem/`) via RemFS abstraction
- [x] **Queue persistence** with ISO timestamp-based files
- [x] **Content file management** in `~/.config/rem/content/`
- [x] **Auto-cleanup** when exceeding 20 items
- [x] **fs.FS abstraction** for testability (RemFS package)
- [x] **Stream-based content model** (`io.ReadSeekCloser`)
- [x] **Metadata extraction** (timestamp, preview generation)
- [x] **Integration testing** with TUI

#### Implemented Queue Manager API:
```go
type QueueManager struct {
    Enqueue(content io.ReadSeekCloser) (*QueueItem, error)
    Get(index int) (*QueueItem, error)
    Size() int
    List() []*QueueItem
    FileSystem() FileSystem
}
```

#### Current Capabilities:
```bash
go run ./cmd/demo/           # Add sample content to queue
go run . view                # Launch TUI with real queue data
go run ./cmd/test-integration/  # Verify TUI integration
```

---

### 🚧 Phase 3: Full CLI Interface (IN PROGRESS)
**Goal**: Complete all `rem store` and `rem get` commands

#### 3.1 Store Operations
- [ ] **Stdin input**: `rem store` (pipe input)
- [ ] **File input**: `rem store file.txt`
- [ ] **Clipboard input**: `rem store -c`
- [ ] **Input validation** and error handling
- [ ] **Content type detection**

#### 3.2 Get Operations
- [ ] **Stdout output**: `rem get N`
- [ ] **Clipboard output**: `rem get -c N`
- [ ] **File output**: `rem get N file.txt`
- [ ] **Interactive viewer**: `rem get` (integrate existing TUI)
- [ ] **Error handling** for invalid indices

#### 3.3 Integration Testing
```bash
# Full workflow testing
echo "test data" | rem store
rem store ~/.bashrc
rem store -c

rem get                       # Browse all items
rem get 0                     # Output first item
rem get -c 1                  # Copy second item to clipboard
rem get 2 output.txt         # Save third item to file
```

**Estimated Time**: 1-2 weeks

---

### 🔮 Phase 4: Polish & Advanced Features
**Goal**: Production-ready features and user experience

#### 4.1 Configuration System
- [ ] **Config file** (`~/.config/rem/config.toml`)
- [ ] **Queue size limits** (default 20 items)
- [ ] **Content size limits**
- [ ] **Auto-cleanup policies**

#### 4.2 Enhanced TUI Features
- [ ] **Item deletion** from queue
- [ ] **Copy operations** within TUI
- [ ] **Multiple selection** support
- [ ] **Export operations**
- [ ] **Themes and customization**

#### 4.3 Advanced Content Handling
- [ ] **Binary content** detection and handling
- [ ] **Image preview** support
- [ ] **Syntax highlighting** for code content
- [ ] **Large file** handling optimizations

#### 4.4 Quality & Distribution
- [ ] **Comprehensive tests** (unit, integration, TUI)
- [ ] **Documentation** (man page, --help, examples)
- [ ] **Installation scripts**
- [ ] **CI/CD pipeline**
- [ ] **Release packaging**

**Estimated Time**: 2-3 weeks

---

### 🌟 Phase 5: Extended Features (FUTURE)
**Goal**: Advanced functionality for power users

#### Potential Features:
- [ ] **Encryption** for sensitive content
- [ ] **Network sources** (HTTP/HTTPS content)
- [ ] **Multi-device sync**
- [ ] **Plugin system** for content processors
- [ ] **Full-text search** with indexing
- [ ] **Scripting integration** (rem as library)
- [ ] **Shell integration** (completion, aliases)

---

## Technical Milestones

### Milestone 1: ✅ Working TUI (DONE)
- Interactive dual-pane viewer
- Search functionality
- Stream-based architecture
- Clean code organization

### Milestone 2: ✅ Core Storage System (DONE)
- File-based queue persistence
- fs.FS abstraction for testability
- ISO timestamp-based content files
- Auto-cleanup and size management
- Full TUI integration with real data

### Milestone 3: 🎯 MVP CLI Tool (NEXT)
- Basic store/get operations
- Clipboard integration
- **Target**: Usable daily clipboard manager

### Milestone 3: 🏁 Production Release
- Full CLI interface
- Robust error handling
- Documentation and tests
- **Target**: Public release ready

### Milestone 4: 🚀 Advanced Features
- Enhanced UX and power features
- Extended content support
- Integration ecosystem

---

## Development Priorities

### High Priority (Must Have)
1. **Queue persistence** - Core functionality
2. **Basic store/get operations** - MVP requirement
3. **Clipboard integration** - Essential for daily use
4. **Error handling** - Reliability and user experience

### Medium Priority (Should Have)
1. **Configuration system** - Customization and limits
2. **Advanced TUI features** - Enhanced usability
3. **Content type handling** - Better user experience
4. **Documentation** - Adoption and maintenance

### Low Priority (Nice to Have)
1. **Binary content support** - Niche use cases
2. **Syntax highlighting** - Visual enhancement
3. **Network integration** - Advanced workflows
4. **Plugin system** - Extensibility

---

## Risk Assessment & Mitigation

### Technical Risks
1. **Clipboard Integration Complexity**
   - **Risk**: Platform-specific clipboard APIs
   - **Mitigation**: Use proven Go libraries (like `golang.design/x/clipboard`)

2. **File System Permissions**
   - **Risk**: Config directory access issues
   - **Mitigation**: Graceful fallback, clear error messages

3. **Memory Usage with Large Content**
   - **Risk**: Loading large files into memory
   - **Mitigation**: Stream-based architecture already in place

### Product Risks
1. **User Adoption**
   - **Risk**: Complex interface for simple clipboard needs
   - **Mitigation**: Focus on intuitive defaults, extensive documentation

2. **Platform Compatibility**
   - **Risk**: macOS/Linux/Windows differences
   - **Mitigation**: Start with macOS, expand iteratively

---

## Success Metrics

### Phase 2 Success Criteria ✅
- [x] Persistent queue survives application restart
- [x] TUI displays real stored content (not dummy data)
- [x] fs.FS abstraction enables clean testing
- [x] Auto-cleanup maintains queue size limits

### Phase 3 Success Criteria
- [ ] Complete CLI interface matches specification
- [ ] Can replace basic pbcopy/pbpaste workflows
- [ ] Error handling covers common edge cases
- [ ] Performance suitable for daily use

### Overall Project Success
- [ ] Daily usable clipboard manager
- [ ] Superior to basic clipboard tools
- [ ] Extensible architecture for future features
- [ ] Clean, maintainable codebase

---

## Next Steps (Immediate Actions)

### Week 1: CLI Commands (CURRENT FOCUS)
1. **Implement `rem store` command with stdin/file/clipboard input**
2. **Add `rem get N` for stdout output operations**
3. **Integrate clipboard support**
4. **Command-line argument parsing and validation**

### Week 2: Polish & Integration
1. **End-to-end testing of store → view → get workflow**
2. **Error handling and edge cases**
3. **Performance optimization**
4. **Basic documentation and help system**

### Week 3: Polish & Testing
1. **Error handling and edge cases**
2. **Clipboard integration testing**
3. **Performance optimization**
4. **Documentation updates**

This plan provides a clear roadmap from the current TUI foundation to a complete, production-ready clipboard manager.