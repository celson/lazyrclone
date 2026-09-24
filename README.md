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
- Go 1.22+ (to build from source)
- [rclone](https://rclone.org/downloads/) installed and on `$PATH` *(optional: runs in demo mode if missing)*

### Installation

```bash
git clone https://github.com/celson/lazyrclone.git
cd lazyrclone
make build
./bin/lazyrclone
```

Or install directly with Go:
```bash
go install ./cmd/lazyrclone
```

### Running Demo Mode
```bash
lazyrclone --demo
```

---

## ⌨️ Keybindings

### Global
| Key | Action |
| --- | --- |
| `1` | Switch to **Profiles** tab |
| `2` | Switch to **Explorer** tab |
| `3` | Switch to **Transfers** tab |
| `4` | Switch to **Remotes** tab |
| `Tab` | Switch between Views / Explorer Panes |
| `?` | Toggle Help dialog |
| `q` / `Ctrl+C` | Quit |

### Profiles View
| Key | Action |
| --- | --- |
| `d` | Run **honest dry-run diff** preview |
| `r` / `Enter` | Execute selected profile |
| `n` | Create a new profile |
| `e` | Edit selected profile |
| `x` | Delete selected profile |
| `f` | Focus diff preview (scroll with `j`/`k`) |

### Dual-Pane Explorer View
| Key | Action |
| --- | --- |
| `Tab` / `h` / `l` | Switch active pane (Left ⟷ Right) |
| `Enter` | Enter directory |
| `Backspace` / `-` | Go to parent directory |
| `Space` | Select / deselect file |
| `r` | Change remote for active pane (e.g., `gdrive:`, `s3:`, `local:`) |
| `c` | Copy selected item(s) to the target pane |
| `s` | Sync active directory to the target pane |
| `n` | Create a new directory |
| `x` | Delete file / directory (with confirmation) |
| `R` | Refresh current pane |

### Transfers View
| Key | Action |
| --- | --- |
| `j` / `k` | Select transfer job |
| `x` | Cancel / stop active job |
| `c` | Clear completed transfer jobs |

### Remotes View
| Key | Action |
| --- | --- |
| `j` / `k` | Select remote |
| `t` | Test connection / reachability |
| `r` | Refresh remotes and storage quota |

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
