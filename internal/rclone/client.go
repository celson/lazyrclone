package rclone

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RcloneClient interface {
	IsAvailable() bool
	IsMock() bool
	Version() (string, error)
	ListRemotes(ctx context.Context) ([]RemoteInfo, error)
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

func (c *RealClient) IsAvailable() bool {
	_, err := exec.LookPath(c.binaryPath)
	return err == nil
}

func (c *RealClient) IsMock() bool {
	return false
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
					detailMap[k] = fmt.Sprintf("%v", v)
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
	// If path is local without prefix or with "local:", list directory locally if rclone fails
	if strings.HasPrefix(remotePath, "local:") {
		localPath := strings.TrimPrefix(remotePath, "local:")
		if localPath == "" {
			localPath = "."
		}
		if strings.HasPrefix(localPath, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				localPath = filepath.Join(home, strings.TrimPrefix(localPath, "~"))
			}
		}
		return listLocalDir(localPath)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, "lsjson", remotePath, "--max-depth", "1")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsjson error: %w", err)
	}

	var items []FileItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("failed to parse lsjson: %w", err)
	}

	return items, nil
}

func listLocalDir(dirPath string) ([]FileItem, error) {
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
	cmd := exec.CommandContext(ctx, c.binaryPath, "mkdir", remotePath)
	return cmd.Run()
}

func (c *RealClient) Delete(ctx context.Context, remotePath string, isDir bool) error {
	var cmd *exec.Cmd
	if isDir {
		cmd = exec.CommandContext(ctx, c.binaryPath, "purge", remotePath)
	} else {
		cmd = exec.CommandContext(ctx, c.binaryPath, "deletefile", remotePath)
	}
	return cmd.Run()
}

func (c *RealClient) DryRun(ctx context.Context, op string, src string, dest string, flags []string) (*DryRunResult, error) {
	start := time.Now()
	args := []string{op, src, dest, "--dry-run", "-v", "--use-json-log"}
	args = append(args, flags...)

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

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
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

	_ = cmd.Wait()
	result.Duration = time.Since(start)
	return result, nil
}

func (c *RealClient) StartTransfer(ctx context.Context, job *TransferJob, onStats func(*StatsMsg), onLog func(string)) error {
	args := []string{
		job.Operation,
		job.Source,
		job.Destination,
		"--stats", "500ms",
		"--stats-log-level", "NOTICE",
		"--use-json-log",
	}

	if job.IsDryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		_, stats, msg := ParseLogLine(line)
		if stats != nil && onStats != nil {
			onStats(stats)
		}
		if msg != "" && onLog != nil {
			onLog(msg)
		}
	}

	return cmd.Wait()
}
