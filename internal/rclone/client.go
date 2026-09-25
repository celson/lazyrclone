package rclone

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

func isValidOperation(op string) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "copy", "sync", "move", "bisync", "check":
		return true
	default:
		return false
	}
}

type RcloneClient interface {
	IsAvailable() bool
	IsMock() bool
	BinaryPath() string
	Version() (string, error)
	ListRemotes(ctx context.Context) ([]RemoteInfo, error)
	DeleteRemote(ctx context.Context, name string) error
	AboutRemote(ctx context.Context, remote string) (*AboutInfo, error)
	ListDir(ctx context.Context, remotePath string) ([]FileItem, error)
	CreateDir(ctx context.Context, remotePath string) error
	Delete(ctx context.Context, remotePath string, isDir bool) error
	DryRun(ctx context.Context, op string, src string, dest string, flags []string) (*DryRunResult, error)
	StartTransfer(ctx context.Context, job *TransferJob, onStats func(*StatsMsg), onLog func(string)) error
}

type RealClient struct {
	binaryPath string
}

func NewRealClient(binaryPath string) *RealClient {
	if binaryPath == "" {
		binaryPath = "rclone"
	}
	return &RealClient{binaryPath: binaryPath}
}

// NormalizeRclonePath converts paths like "local:~/Documents" or "~/Documents"
// into normalized paths that rclone CLI understands (/home/user/Documents).
func NormalizeRclonePath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "local:") {
		p = strings.TrimPrefix(p, "local:")
		if p == "" {
			p = "."
		}
	}
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func (c *RealClient) IsAvailable() bool {
	_, err := exec.LookPath(c.binaryPath)
	return err == nil
}

func (c *RealClient) IsMock() bool {
	return false
}

func (c *RealClient) BinaryPath() string {
	if c.binaryPath == "" {
		return "rclone"
	}
	return c.binaryPath
}

func (c *RealClient) DeleteRemote(ctx context.Context, name string) error {
	clean := strings.TrimSuffix(name, ":")
	cmd := exec.CommandContext(ctx, c.binaryPath, "config", "delete", clean)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete remote %s: %w (%s)", clean, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *RealClient) Version() (string, error) {
	cmd := exec.Command(c.binaryPath, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "rclone", nil
}

func (c *RealClient) ListRemotes(ctx context.Context) ([]RemoteInfo, error) {
	// First try config dump to get types as well
	cmd := exec.CommandContext(ctx, c.binaryPath, "config", "dump")
	out, err := cmd.Output()
	if err == nil {
		var dumped map[string]map[string]interface{}
		if err := json.Unmarshal(out, &dumped); err == nil {
			remotes := make([]RemoteInfo, 0, len(dumped)+1)
			for name, details := range dumped {
				t := "unknown"
				if val, ok := details["type"].(string); ok {
					t = val
				}
				detailMap := make(map[string]string)
				for k, v := range details {
					if IsSensitiveConfigKey(k) {
						detailMap[k] = "[REDACTED]"
					} else {
						detailMap[k] = fmt.Sprintf("%v", v)
					}
				}
				remotes = append(remotes, RemoteInfo{
					Name:    name + ":",
					Type:    t,
					Details: detailMap,
				})
			}
			// Always append local
			remotes = append(remotes, RemoteInfo{
				Name: "local:",
				Type: "local",
			})
			return remotes, nil
		}
	}

	// Fallback to listremotes
	cmdList := exec.CommandContext(ctx, c.binaryPath, "listremotes")
	outList, err := cmdList.Output()
	if err != nil {
		return []RemoteInfo{{Name: "local:", Type: "local"}}, nil
	}

	lines := strings.Split(string(outList), "\n")
	var remotes []RemoteInfo
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			remotes = append(remotes, RemoteInfo{
				Name: l,
				Type: "remote",
			})
		}
	}
	remotes = append(remotes, RemoteInfo{Name: "local:", Type: "local"})
	return remotes, nil
}

func (c *RealClient) AboutRemote(ctx context.Context, remote string) (*AboutInfo, error) {
	remote = strings.TrimSuffix(remote, "/")
	if !strings.HasSuffix(remote, ":") && !strings.Contains(remote, ":") {
		remote = remote + ":"
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, "about", remote, "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	start := strings.Index(string(out), "{")
	end := strings.LastIndex(string(out), "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no json found in about output")
	}

	var info AboutInfo
	if err := json.Unmarshal(out[start:end+1], &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *RealClient) ListDir(ctx context.Context, remotePath string) ([]FileItem, error) {
	norm := NormalizeRclonePath(remotePath)
	if !strings.Contains(norm, ":") {
		return listLocalDir(norm)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, "lsjson", norm, "--max-depth", "1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		var exitErr *exec.ExitError
		if (errors.As(err, &exitErr) && exitErr.ExitCode() == 3) || strings.Contains(strings.ToLower(errStr), "directory not found") {
			return nil, fmt.Errorf("directory not found: '%s' does not exist yet", norm)
		}
		if errStr != "" {
			lines := strings.Split(errStr, "\n")
			for _, l := range lines {
				if strings.Contains(l, "ERROR") {
					return nil, fmt.Errorf("%s", strings.TrimSpace(l))
				}
			}
			return nil, fmt.Errorf("%s", strings.TrimSpace(lines[len(lines)-1]))
		}
		return nil, fmt.Errorf("lsjson error: %w", err)
	}

	var items []FileItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("failed to parse lsjson: %w", err)
	}

	return items, nil
}

func listLocalDir(dirPath string) ([]FileItem, error) {
	if dirPath == "" || dirPath == "." {
		var err error
		dirPath, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var items []FileItem
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, FileItem{
			Path:    entry.Name(),
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}
	return items, nil
}

func (c *RealClient) CreateDir(ctx context.Context, remotePath string) error {
	norm := NormalizeRclonePath(remotePath)
	if !strings.Contains(norm, ":") {
		return os.MkdirAll(norm, 0755)
	}
	cmd := exec.CommandContext(ctx, c.binaryPath, "mkdir", norm)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create directory: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *RealClient) Delete(ctx context.Context, remotePath string, isDir bool) error {
	norm := NormalizeRclonePath(remotePath)
	clean := filepath.Clean(norm)

	// Security guard: prevent accidental deletion of root or current directory
	if clean == "/" || clean == "." || clean == "" || norm == "" || norm == "/" {
		return fmt.Errorf("refusing to delete root path %q", norm)
	}

	// Security guard: protect user home directory and key system directories
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cleanHome := filepath.Clean(home)
		if clean == cleanHome {
			return fmt.Errorf("refusing to delete user home directory %q", norm)
		}
	}
	protectedDirs := []string{"/home", "/root", "/etc", "/usr", "/var", "/bin", "/sbin", "/lib", "/boot", "/dev", "/sys", "/proc"}
	for _, p := range protectedDirs {
		if clean == p {
			return fmt.Errorf("refusing to delete protected system path %q", norm)
		}
	}

	// Security guard for remotes: prevent purging entire remote root (e.g. "gdrive:", "gdrive:/")
	trimmed := strings.TrimSpace(norm)
	if strings.HasSuffix(trimmed, ":") || strings.HasSuffix(trimmed, ":/") || strings.HasSuffix(trimmed, ":.") || strings.HasSuffix(trimmed, ":./") {
		return fmt.Errorf("refusing to delete root of remote %q", norm)
	}

	if !strings.Contains(norm, ":") {
		if isDir {
			return os.RemoveAll(norm)
		}
		return os.Remove(norm)
	}

	var cmd *exec.Cmd
	if isDir {
		cmd = exec.CommandContext(ctx, c.binaryPath, "purge", norm)
	} else {
		cmd = exec.CommandContext(ctx, c.binaryPath, "deletefile", norm)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete %q: %w (%s)", norm, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *RealClient) DryRun(ctx context.Context, op string, src string, dest string, flags []string) (*DryRunResult, error) {
	if !isValidOperation(op) {
		return nil, fmt.Errorf("unsupported or invalid rclone operation: %q", op)
	}
	start := time.Now()
	cleanSrc := NormalizeRclonePath(src)
	cleanDest := NormalizeRclonePath(dest)

	args := []string{op, cleanSrc, cleanDest, "--dry-run", "-v", "--use-json-log"}
	for _, f := range flags {
		for _, part := range strings.Fields(f) {
			if strings.TrimSpace(part) != "" {
				args = append(args, part)
			}
		}
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	result := &DryRunResult{
		Items: make([]DryRunItem, 0),
	}

	var errMessages []string

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `"level":"critical"`) || strings.Contains(line, `"level":"error"`) {
			var raw struct {
				Msg string `json:"msg"`
			}
			if jsonErr := json.Unmarshal([]byte(line), &raw); jsonErr == nil && raw.Msg != "" {
				clean := strings.TrimSpace(stripAnsi(raw.Msg))
				if clean != "" {
					errMessages = append(errMessages, clean)
				}
			} else {
				clean := strings.TrimSpace(stripAnsi(line))
				if clean != "" {
					errMessages = append(errMessages, clean)
				}
			}
		}

		item, _, _ := ParseLogLine(line)
		if item != nil {
			result.Items = append(result.Items, *item)
			switch item.Action {
			case ActionAdd:
				result.ToAdd++
			case ActionUpdate:
				result.ToUpdate++
			case ActionDelete:
				result.ToDelete++
			default:
				result.Unchanged++
			}
		}
	}

	waitErr := cmd.Wait()
	result.Duration = time.Since(start)

	if waitErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Err = fmt.Errorf("dry-run timed out after %v (tip: remove '--fast-list' if using Google Drive/Dropbox)", time.Since(start).Round(time.Second))
		} else if errors.Is(ctx.Err(), context.Canceled) {
			result.Err = fmt.Errorf("dry-run cancelled")
		} else if len(errMessages) > 0 {
			result.Err = fmt.Errorf("%s", strings.Join(errMessages, " | "))
		} else {
			result.Err = waitErr
		}
		return result, result.Err
	}

	return result, nil
}

// BuildTransferArgs constructs the command-line arguments for rclone from a TransferJob
func BuildTransferArgs(job *TransferJob) []string {
	cleanSrc := NormalizeRclonePath(job.Source)
	cleanDest := NormalizeRclonePath(job.Destination)

	args := []string{
		job.Operation,
		cleanSrc,
		cleanDest,
		"--stats", "500ms",
		"--stats-log-level", "NOTICE",
		"--use-json-log",
	}

	if job.IsDryRun {
		args = append(args, "--dry-run")
	}

	if job.Transfers > 0 {
		args = append(args, "--transfers", strconv.Itoa(job.Transfers))
	}
	if job.Checkers > 0 {
		args = append(args, "--checkers", strconv.Itoa(job.Checkers))
	}
	for _, exc := range job.Exclude {
		if strings.TrimSpace(exc) != "" {
			args = append(args, "--exclude", exc)
		}
	}
	for _, f := range job.Flags {
		for _, part := range strings.Fields(f) {
			if strings.TrimSpace(part) != "" {
				args = append(args, part)
			}
		}
	}
	return args
}

func (c *RealClient) StartTransfer(ctx context.Context, job *TransferJob, onStats func(*StatsMsg), onLog func(string)) error {
	if !isValidOperation(job.Operation) {
		return fmt.Errorf("unsupported or invalid rclone operation: %q", job.Operation)
	}

	args := BuildTransferArgs(job)

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var errMessages []string

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `"level":"critical"`) || strings.Contains(line, `"level":"error"`) {
			var raw struct {
				Msg string `json:"msg"`
			}
			if jsonErr := json.Unmarshal([]byte(line), &raw); jsonErr == nil && raw.Msg != "" {
				clean := strings.TrimSpace(stripAnsi(raw.Msg))
				if clean != "" {
					errMessages = append(errMessages, clean)
				}
			} else {
				clean := strings.TrimSpace(stripAnsi(line))
				if clean != "" {
					errMessages = append(errMessages, clean)
				}
			}
		}

		_, stats, msg := ParseLogLine(line)
		if stats != nil && onStats != nil {
			onStats(stats)
		}
		if msg != "" && onLog != nil {
			onLog(msg)
		}
	}

	waitErr := cmd.Wait()
	if waitErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("transfer timed out")
		} else if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("transfer cancelled")
		}
		if len(errMessages) > 0 {
			return fmt.Errorf("%s", strings.Join(errMessages, " | "))
		}
		return waitErr
	}
	return nil
}
