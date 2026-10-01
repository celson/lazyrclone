# lazyrclone ⚡

A terminal user interface (TUI) for [rclone](https://rclone.org/), inspired by [lazyrsync](https://github.com/westpoint-io/lazyrsync) and [lazygit](https://github.com/jesseduffield/lazygit).

Built in **Go** using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## ✨ Features

- **Honest Dry-Run Previews**: Preview additions (`+`), updates (`~`), deletions (`-`), and identical files (`=`) before performing any transfer.
- **Reusable Profiles**: Save and run frequent Sync, Copy, Move, or Check tasks between local directories and cloud providers.
- **Dual-Pane Remote Explorer**: Browse remote and local filesystems side-by-side with intuitive directory navigation, multi-selection, and one-key transfers.
- **Real-Time Transfer Progress**: Live progress bars, transfer speeds, ETA, active file streams, and live logs.
- **Remotes & Quota Inspection**: View configured remotes, test reachability, and monitor storage quota (`rclone about`).
- **Safety Safeguards**: Confirmation modals for destructive actions (`--delete`, `sync`, `purge`, file removal).
- **Built-in Demo Mode**: Instant preview and simulation if `rclone` is not installed or configured yet.

---

## 🚀 Quick Start

### Prerequisites
- [mise](https://mise.jdx.dev/) or Go 1.22+ (to build from source)
- [rclone](https://rclone.org/downloads/) installed and on `$PATH` *(optional: runs in demo mode if missing)*

### Installation & Build

Using **[mise](https://mise.jdx.dev/)**:
```bash
git clone https://github.com/celson/lazyrclone.git
cd lazyrclone

# Build binary into ./bin/lazyrclone
mise run build

# Or directly run lazyrclone
mise run run

# Run unit tests
mise run test
```

Or directly with Go:
```bash
go install ./cmd/lazyrclone
```

### Running Demo Mode
```bash
mise run demo
# or: ./bin/lazyrclone --demo
```

---

## ⌨️ Keybindings

### Global Panel Navigation
| Key | Action |
| --- | --- |
| `1` | Focus **[1] Profiles** panel |
| `2` | Focus **[2] Transfers & Runs** panel |
| `3` | Focus **[3] Main View** panel |
| `Tab` / `Shift+Tab` | Cycle focus forward / backward between panels |
| `[` / `]` | Switch Main View sub-tab (**Diff Preview** ⟷ **Dual Explorer** ⟷ **Live Logs**) |
| `?` | Toggle interactive Help overlay |
| `q` / `Ctrl+C` | Quit |

### Panel [1] Profiles (Tasks & Remotes)
| Key | Action |
| --- | --- |
| `j` / `k` or `↑` / `↓` | Navigate profiles list |
| `d` or `p` | Run **honest dry-run diff** (renders live in Main View!) |
| `r` / `Enter` | Execute sync/copy operation with live progress |
| `n` | Create a new profile |
| `e` | Edit selected profile |
| `x` | Delete selected profile |

### Panel [2] Transfers & Runs
| Key | Action |
| --- | --- |
| `j` / `k` | Select transfer run |
| `x` | Cancel / stop active job |
| `c` | Clear completed jobs |

### Panel [3] Main View (Diff / Explorer / Logs)
| Key | Action |
| --- | --- |
| `[` / `]` | Toggle sub-tab (**Diff** \| **Explorer** \| **Logs**) |
| `j` / `k` | Scroll diff lines or job logs |
| `←` / `→` | Switch active pane in Dual Explorer (Source ⟷ Dest) |
| `Enter` | Open directory in Explorer |
| `Backspace` / `-` | Go to parent directory |
| `Space` | Select / deselect file in Explorer |
| `c` | Copy selected item(s) to target pane |
| `s` | Sync directory to target pane |
| `n` | Create new directory in Explorer |
| `x` | Delete file / directory (with confirmation) |

---

## ⚙️ Configuration

`lazyrclone` saves profiles and settings to:
`~/.config/lazyrclone/config.yaml`

Example:
```yaml
settings:
  confirm_destructive: true
  default_view: profiles
  theme: default
  rclone_path: rclone

profiles:
  - id: backup-documents
    name: Backup Documents -> GDrive
    source: local:~/Documents
    destination: gdrive:Backup/Documents
    operation: copy
    flags:
      - --fast-list
    exclude:
      - "*.tmp"
      - "node_modules/**"
    transfers: 4
    checkers: 8
```

---

## 📄 License
MIT
