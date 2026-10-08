# lazyrclone

A terminal UI for [rclone](https://rclone.org/), inspired by [lazyrsync](https://github.com/westpoint-io/lazyrsync). Written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

lazyrclone does not reimplement any transfer logic: it runs the `rclone` binary on your `$PATH` (or the one you point it to) and shows the result.

## Features

- **Profiles**: saved source/destination pairs with an operation (`copy`, `sync`, `move`, `bisync` or `check`), extra rclone flags, exclude patterns, and `--transfers` / `--checkers` values. Create, edit, delete and run them from the UI.
- **Dry-run preview**: runs the operation with `--dry-run` and lists each file as ADD, UPDATE, DELETE or EQUAL, with totals.
- **Live transfers**: a run is started with `--use-json-log` and stats every 500 ms. The Transfers panel shows progress, and Live Logs shows the output. Jobs can be stopped.
- **Dual-pane explorer**: browse two locations side by side (local or any rclone remote), mark items, copy to the other pane, sync the current folder to the other pane, create folders and delete.
- **Remotes**: lists the remotes from `rclone config dump` / `rclone listremotes`, tests reachability (by listing the remote), shows quota from `rclone about --json`, and can launch `rclone config`, `rclone config reconnect <remote>` or delete a remote. Sensitive config keys (passwords, tokens, keys, etc.) are filtered out of the details shown.
- **Confirmations**: running a profile whose operation is `sync`, `move` or `bisync` asks for confirmation (controlled by `confirm_destructive`). Syncing from the explorer, deleting files/folders, deleting profiles and deleting remotes always ask.
- **Delete guards**: the explorer refuses to delete `/`, the home directory, a set of system directories (`/home`, `/etc`, `/usr`, ...) and the root of a remote.
- **Demo mode**: with `--demo`, or automatically when the `rclone` binary cannot be found, lazyrclone uses a simulated client with sample remotes and data instead of calling rclone.

## Installation

### Release binaries

Pre-built archives for Linux and macOS (amd64 and arm64) are attached to each [GitHub release](https://github.com/celson/lazyrclone/releases), together with a `checksums.txt`.

### With mise

```bash
mise use -g github:celson/lazyrclone
```

This downloads the matching release archive from GitHub. To pin a version: `mise use -g github:celson/lazyrclone@0.1.0`.

### From source

Requires the Go version declared in `go.mod` (currently 1.26.8). The repo pins it for [mise](https://mise.jdx.dev/) in `mise.toml`.

```bash
git clone https://github.com/celson/lazyrclone.git
cd lazyrclone

mise run build      # builds ./bin/lazyrclone with version info
# or, with plain Go:
go build -o bin/lazyrclone ./cmd/lazyrclone
```

Other mise tasks: `run`, `demo`, `test`, `test:race`, `install` (go install with version flags) and `clean`.

## Usage

```bash
lazyrclone                       # use the real rclone
lazyrclone --demo                # simulated mode, no rclone needed
lazyrclone --rclone-path /path/to/rclone
lazyrclone --version
```

`rclone` is only required outside demo mode. If `--rclone-path` is left at its default, the `rclone_path` setting from the config file is used.

## Interface

Four panels, plus a main view with four tabs:

| Panel | Key |
| --- | --- |
| Profiles | `1` |
| Remotes | `2` |
| Transfers | `3` |
| Main view | `4` |

Main view tabs: **Diff Preview**, **Dual Explorer**, **Live Logs**, **Remotes**. Selecting panel `1`, `2` or `3` also switches the main view to Diff Preview, Remotes or Live Logs respectively.

## Keybindings

### Global
| Key | Action |
| --- | --- |
| `1` `2` `3` `4` | Focus panel |
| `Tab` / `Shift+Tab` | Next / previous panel |
| `[` `,` / `]` `.` | Previous / next main view tab |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

### Profiles
| Key | Action |
| --- | --- |
| `j` `k` / `↑` `↓` | Move selection |
| `d` / `p` | Dry-run the selected profile |
| `Enter` / `r` | Run the selected profile |
| `s` | Stop the profile's running job (or the only running job, or a running dry-run) |
| `n` | New profile |
| `e` | Edit profile (name, source, destination, operation, flags) |
| `x` | Delete profile |
| `l` / `→` | Move focus to the main view |
| `R` | Go to the Remotes panel |

### Remotes
| Key | Action |
| --- | --- |
| `j` `k` / `↑` `↓` | Move selection |
| `t` | Test the selected remote |
| `r` | Refresh the list |
| `c` / `n` | Run `rclone config` |
| `e` | Run `rclone config reconnect` for the selected remote |
| `d` / `x` | Delete the selected remote |
| `Enter` | Open the remote in the active explorer pane |

### Transfers
| Key | Action |
| --- | --- |
| `j` `k` / `↑` `↓` | Move selection |
| `Enter` / `l` / `→` | Show the job's logs |
| `s` / `x` | Stop the selected job |
| `c` | Clear finished jobs |

### Dual Explorer (main view)
| Key | Action |
| --- | --- |
| `←` / `→` | Switch active pane |
| `j` `k` / `↑` `↓` | Move selection |
| `Enter` | Open directory |
| `Backspace` / `-` | Parent directory |
| `Space` | Mark / unmark item |
| `r` | Change the pane's remote |
| `R` | Reload the pane |
| `c` | Copy to the other pane (starts a job for the first selected or marked item) |
| `s` | Sync the current folder to the other pane (asks for confirmation) |
| `n` | Create a directory |
| `x` | Delete the selected or marked items (asks for confirmation) |

### Diff Preview and Live Logs (main view)
| Key | Action |
| --- | --- |
| `j` `k` / `↑` `↓` | Scroll |
| `d` / `p` | Re-run the dry-run (Diff Preview) |
| `s` / `x` | Stop the dry-run (Diff Preview) or the selected job (Live Logs) |
| `Esc` / `h` / `←` | Return to the previously focused side panel |

## Configuration

Profiles and settings live in `~/.config/lazyrclone/config.yaml`. Set `LAZYRCLONE_CONFIG_DIR` to use another directory. On first run, if no file exists, one is created with three sample profiles, which you will likely want to edit or delete. The file is written with mode `0600`.

```yaml
settings:
  confirm_destructive: true
  default_view: profiles
  theme: catppuccin-mocha
  rclone_path: rclone

profiles:
  - id: backup-documents
    name: Backup Documents -> GDrive
    source: local:~/Documents
    destination: gdrive:Backup/Documents
    operation: copy          # copy | sync | move | bisync | check
    flags:
      - --fast-list
    exclude:
      - "*.tmp"
      - "node_modules/**"
    transfers: 4
    checkers: 8
```

Paths may be written as `local:~/dir` or `~/dir`; `local:` and a leading `~` are expanded before calling rclone.

Current limitations of the config file:
- `include` and `bandwidth_limit` exist in the profile format but are not passed to rclone.
- `theme` and `default_view` are stored but not read by the UI.
- The in-app profile editor only edits name, source, destination, operation and flags. `exclude`, `transfers` and `checkers` have to be edited in the YAML file.
- Dry-runs apply a profile's `exclude` and `flags`; `transfers` and `checkers` are only used for real runs.

## Development

```bash
mise run test        # go test -v ./...
mise run test:race   # with the race detector
```

CI (`.github/workflows/ci.yml`) runs `go vet` and `go test -race` on Linux and macOS, and does a GoReleaser snapshot build. Pushing a `v*` tag runs the release workflow, which builds the archives with GoReleaser and publishes a GitHub release.

## License

No license file has been added to the repository yet.
