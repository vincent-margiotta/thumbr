# Thumbr CLI Project Requirements Document (Final Revision)

## 1. Introduction

**Project Name:** Thumbr
**Binary Name:** `thumbr`
**Runtime:** Go (Golang)
**Framework:** Bubble Tea (TUI)
**Goal:** To create a highly efficient, production-ready Command Line Interface (CLI) application that offers a unique, **tactile, and serendipitous browsing experience** for flat-file notes, replicating the feeling of rapidly thumbing through a physical card index.

---

## 2. Functional Requirements (FR)

These define what the application *must do* for the user.

### 2.1 Core File & Data Management

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **FR01** | **Root Configuration** | Define a single root directory (the "box") containing notes (positional arg or config). | Done |
| **FR02** | **File Loading** | Recursively load files matching configurable extensions (default `.txt`) on startup. | Done |
| **FR03** | **Data Structure** | Store files as `Card` objects, ordered by the configured sort rules. | Done |
| **FR04** | **Ignore List** | Respect ignore globs to exclude files/directories from the index. | Done |
| **FR05** | **Real-time Reload** | Detect box changes (new/deleted/renamed) and reload without restarting. | Not started |
| **FR06** | **File Opening** | From the overlay, open the source file in `$EDITOR` or system default. | Done |

### 2.2 Navigation and Browsing (The Analog Experience)

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **FR07** | **Basic Movement** | Navigate the stack with single keypresses (`j/k` or arrows) one card at a time. | Done |
| **FR08** | **Velocity Skipping** | Rapid presses accelerate movement with configurable timing/step limits. | Done |
| **FR09** | **Random Jump (Serendipity)** | Jump to a random note with a single keypress. | Done |
| **FR10** | **Marking/Filtering** | Mark the current card; toggle a marked-only view for quick review; marks/filter state persist across box switches within a session. | Done |
| **FR11** | **Help Overlay** | Show current key bindings via a help overlay. | Done |

### 2.3 Viewing and Rendering

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **FR12** | **Active Stack View** | Render a pseudo-3D stack with a configurable number of receding cards. | Done |
| **FR13** | **Overlay Detail View** | Open an overlay to read the active card’s content. | Done |
| **FR14** | **Plaintext Rendering** | Show file content as-is (no formatting/rendering or link rewriting). | Done |
| **FR15** | **Content Wrapping** | Wrap text to overlay bounds. | Done |
| **FR16** | **Card Flipping/Pagination** | When content exceeds the card height, page through it (e.g., `n/p`) with edge cues to preserve the “flip” feel. | Done |
| **FR17** | **ID Sanitization** | Strip extraneous characters (e.g., brackets) from displayed ID/Title. | Partial |

### 2.4 CLI Behavior and Config

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **FR18** | **Positional Root Path** | Accept an optional positional path to the box root (defaults to CWD). | Done |
| **FR19** | **Alternate Screen Toggle** | Flag to enable/disable the terminal alternate screen. | Done |
| **FR20** | **Random Seed Control** | Flag to set a random seed for reproducible random jumps. | Done |
| **FR21** | **Config File Formats** | `--config` accepts JSON/YAML/TOML; flags override config. | Done |
| **FR22** | **Version Output** | `-v/--version` prints the embedded build version. | Done |
| **FR23** | **Sorting Controls** | Configure lexical vs natural sort, regex for addressed names, and whether matched names appear before or after others. | Done |
| **FR24** | **Performance Benchmarking** | Provide a repeatable benchmark target to measure load/sort performance against the 90k/<2s goal (e.g., `make bench`, configurable note count). | Done |
| **FR25** | **Live Config Reload** | Reload config and apply settings without restarting the app. | Not started |

---

## 3. Non-Functional Requirements (NFR)

These define the quality and operational environment of the application.

### 3.1 Performance and Stability

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **NFR01** | **Startup Speed** | Initial load time for a box containing up to **90,000** files **must** be under 2 seconds. | Validated (warm cache benchmark) |
| **NFR02** | **TUI Responsiveness** | The user interface **must** be immediately responsive to all input and terminal resize events. | Done |
| **NFR03** | **Error Handling** | The application **must** gracefully handle corrupted or unreadable files and display an error message without crashing. | Partial (overlay load surfaced; startup soft) |
| **NFR04** | **Deterministic Randomness** | When a seed is provided, random navigation behavior **must** be deterministic and testable. | Done |

### 3.2 Usability and Configuration

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **NFR05** | **Configuration File** | All configurable parameters **must** be managed via a well-documented configuration file. | Done |
| **NFR06** | **Color Palette** | The application **must** use a focused, configurable color palette. | Done |
| **NFR07** | **Keyboard Mappings** | All primary functions **should** have customizable key bindings in the configuration file. | Done |
| **NFR08** | **Customizable Style Markers** | Border characters/stack spacing **should** be configurable. | Done |

### 3.3 Visual Feedback and Immersion

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **NFR09** | **Velocity Feedback** | The background stack's transition **should** visually change to emphasize fast skips. | Partial (movement accel; no explicit visual delta) |
| **NFR10** | **Filtered View Cue** | When filtering marked cards, provide a persistent visual cue (status indicator and dimming unmarked cards). | Done |
| **NFR11** | **Pagination Edge Cue** | When paging/“flipping,” show a visual cue (e.g., caret/ellipsis) to indicate more content. | Done |

### 3.4 Distribution and Deployment (Cross-Platform Focus)

| ID | Requirement | Description | Status |
| :--- | :--- | :--- | :--- |
| **NFR12** | **Single Binary** | Distribute as a single, statically linked binary for each target OS. | Not started |
| **NFR13** | **Cross-Platform** | Build and test for macOS (ARM/Intel), Linux (amd64), and Windows. | Not started |
| **NFR14** | **Homebrew Support** | Provide a Homebrew Tap for easy installation on macOS and Linux. | Not started |
| **NFR15** | **CI/CD Pipeline** | Automated CI should run tests/lint and generate release binaries. | Not started |
| **NFR16** | **Version Embedding** | Release builds **must** embed a semantic version or `git describe` string via ldflags. | Done |
| **NFR17** | **Terminal Compatibility** | The TUI **must** function correctly across common terminal emulators. | Partial (not validated broadly) |
| **NFR18** | **Benchmark Hygiene** | Benchmarks should clean up temp data or document cleanup steps. | Done (auto-clean) |

---

## 4. Technical Stack (T-Stack)

| Category | Component | Rationale |
| :--- | :--- | :--- |
| **Language** | Go (Golang) | Excellent startup speed and simple static **cross-compilation** (key for Windows support). |
| **TUI Framework** | Bubble Tea (charmbracelet/bubbletea) | Uses the clean Model-View-Update pattern and provides excellent built-in cross-platform terminal handling. |
| **Configuration**| Go's built-in libraries plus a specialized library (e.g., `go-toml/v2`) | Support for mandatory JSON, YAML, and TOML formats. |
| **Packaging** | GoReleaser | Industry standard tool for automating multi-platform binary generation (including Windows). |
