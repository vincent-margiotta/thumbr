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

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **FR01** | **Vault Configuration** | The application **must** allow the user to define a single root directory (the "Vault") containing the notes. |
| **FR02** | **File Loading** | The application **must** recursively load all files matching a configurable extension (default: `.md`) within the Vault upon startup. |
| **FR03** | **Data Structure** | Files **must** be stored internally as `Card` objects, sorted by their unique ID/Title derived from the filename. |
| **FR04** | **Real-time Reload** | The application **should** detect changes in the vault directory (new, deleted, or renamed files) and reload the index without restarting the application. |
| **FR19** | **Ignore List** | The application **must** respect an ignore list (e.g., a config array) to exclude specific files or directories from the index. |
| **FR20** | **File Opening** | From the overlay view, the user **should** be able to open the source file in their system's default text editor (`$EDITOR` or system default). |

### 2.2 Navigation and Browsing (The Analog Experience)

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **FR05** | **Basic Movement** | Users **must** navigate the stack using single keypresses (`j`, `k`, $\downarrow$, $\uparrow$) to move one card at a time. |
| **FR06** | **Velocity Skipping** | The application **must** implement velocity-based movement: rapid, consecutive keypresses (`j`/`k`) must increase the skip distance using configurable timing thresholds. |
| **FR07** | **Random Jump (Serendipity)** | The application **must** allow the user to jump to a random note instantly using a single keypress (`r`). |
| **FR08** | **Approximate GoTo** | **REMOVED** (Breaks the analog illusion). |
| **FR24** | **Card Marking/Filtering** | The user **must** be able to **mark** the current card (`m` key). A separate key (`t` or similar) **must** toggle the browsing mode to filter and only display marked cards, facilitating quick review. |

### 2.3 Viewing and Rendering

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **FR09** | **Active Stack View** | The main browsing screen **must** render a pseudo-3D visual stack of cards, showing a configurable number of cards receding. |
| **FR10** | **Overlay Detail View** | Pressing `Enter` **must** open an overlay window showing the full content of the active card. |
| **FR11** | **Markdown Cleaning** | Content viewing **must** process raw text to: **1)** Remove Obsidian link brackets (`[[link]]` $\rightarrow$ `link`); **2)** **Render** basic styling (e.g., `**bold**` $\rightarrow$ **bold**). |
| **FR12** | **Content Wrapping** | Text in the detail view **must** be dynamically wrapped to fit within the boundaries of the overlay window. |
| **FR18** | **ID Sanitization** | The application **should** strip extraneous characters (e.g., `[`, `]`) from the displayed Card ID/Title. |
| **FR21** | **Seamless Card Flipping** | If content exceeds the card height, dedicated keys **must** turn the card over to the next or previous page of content (discrete pagination) to preserve the physical card constraint. |
| **FR23** | **Help Menu Overlay** | A keypress (`?` or `h`) **must** display a temporary overlay showing the current key bindings and their actions. |

### 2.4 CLI Behavior and Config

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **FR13** | **Positional Vault Path** | CLI **must** accept an optional positional argument for the vault root; defaults to the current working directory. |
| **FR14** | **Alternate Screen Toggle** | CLI **must** expose a flag to enable/disable the terminal alternate screen. |
| **FR15** | **Random Seed Control** | CLI **must** expose a flag to set a random seed for reproducible random jumps. |
| **FR16** | **Config File Formats** | CLI **must** accept a config file path via `--config` and support JSON, YAML, or TOML. Flags override config values. |
| **FR17** | **Version Output** | CLI **must** expose `-v/--version` to print the embedded build version. |

---

## 3. Non-Functional Requirements (NFR)

These define the quality and operational environment of the application.

### 3.1 Performance and Stability

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **NFR01** | **Startup Speed** | Initial load time for a vault containing up to **90,000** files **must** be under 2 seconds. |
| **NFR02** | **TUI Responsiveness** | The user interface **must** be immediately responsive to all input and terminal resize events. |
| **NFR03** | **Error Handling** | The application **must** gracefully handle corrupted or unreadable files and display an error message without crashing. |
| **NFR07** | **Deterministic Randomness** | When a seed is provided, random navigation behavior **must** be deterministic and testable. |

### 3.2 Usability and Configuration

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **NFR04** | **Configuration File** | All configurable parameters **must** be managed via a well-documented configuration file. |
| **NFR05** | **Color Palette** | The application **must** use a focused color palette, with all colors being configurable. |
| **NFR06** | **Keyboard Mappings** | All primary functions **should** have customizable key bindings in the configuration file. |
| **NFR14** | **Customizable Style Markers** | The characters used for borders and stack offset spacing **should** be configurable. |

### 3.3 Visual Feedback and Immersion

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **NFR15** | **Velocity Feedback** | The background stack's transition **should** visually change (e.g., subtle blur or accelerated movement) to emphasize the speed of the skip action (FR06). |
| **NFR16** | **Filtered View Cue** | When in Card Marking/Filtering mode (FR24), the application **must** provide a persistent visual cue (e.g., a status bar indicator AND **visually muting unmarked cards**) to confirm the filter is active. |
| **NFR17** | **Pagination Edge Cue** | When content is being flipped (FR21), the text at the edge of the visible page **should** use a visual cue (e.g., truncation marker or subtle fade) to indicate continuation before the flip is executed. |

### 3.4 Distribution and Deployment (Cross-Platform Focus)

| ID | Requirement | Description |
| :--- | :--- | :--- |
| **NFR08** | **Single Binary** | The application **must** be distributed as a single, statically linked binary for each target OS (Go's primary advantage). |
| **NFR09** | **Cross-Platform** | Binaries **must** be built and tested for **macOS (ARM/Intel), Linux (amd64), and Windows**. |
| **NFR10** | **Homebrew Support** | The project **must** provide a Homebrew Tap for easy installation on macOS and Linux (`brew install thumbr`). |
| **NFR11** | **CI/CD Pipeline** | Automated GitHub Actions **must** handle continuous integration, testing, and generating release binaries. |
| **NFR12** | **Version Embedding** | Release builds **must** embed a semantic version or `git describe` string via ldflags for traceability. |
| **NFR18** | **Terminal Compatibility** | The TUI **must** function correctly across common terminal emulators (e.g., iTerm2, macOS Terminal, Windows Terminal, Kitty, Alacritty). |

---

## 4. Technical Stack (T-Stack)

| Category | Component | Rationale |
| :--- | :--- | :--- |
| **Language** | Go (Golang) | Excellent startup speed and simple static **cross-compilation** (key for Windows support). |
| **TUI Framework** | Bubble Tea (charmbracelet/bubbletea) | Uses the clean Model-View-Update pattern and provides excellent built-in cross-platform terminal handling. |
| **Configuration**| Go's built-in libraries plus a specialized library (e.g., `go-toml/v2`) | Support for mandatory JSON, YAML, and TOML formats. |
| **Packaging** | GoReleaser | Industry standard tool for automating multi-platform binary generation (including Windows). |
