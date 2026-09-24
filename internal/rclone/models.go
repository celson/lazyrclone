package rclone

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// IsSensitiveConfigKey checks if a configuration key represents sensitive credentials
// such as passwords, tokens, API keys, client secrets, etc.
func IsSensitiveConfigKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	sensitiveSubstrings := []string{
		"pass", "password", "secret", "token", "key", "auth",
		"cred", "cert", "hash", "salt", "session", "signature",
		"bearer", "oauth",
	}
	for _, sub := range sensitiveSubstrings {
		if strings.Contains(k, sub) {
			return true
		}
	}
	return false
}

type RemoteInfo struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Details map[string]string `json:"details,omitempty"`
}

type AboutInfo struct {
	Total   int64 `json:"total"`
	Used    int64 `json:"used"`
	Free    int64 `json:"free"`
	Trashed int64 `json:"trashed"`
	Other   int64 `json:"other"`
	Objects int64 `json:"objects"`
}

type FileItem struct {
	Path     string    `json:"Path"`
	Name     string    `json:"Name"`
	Size     int64     `json:"Size"`
	MimeType string    `json:"MimeType"`
	ModTime  time.Time `json:"ModTime"`
	IsDir    bool      `json:"IsDir"`
}

type DryRunAction string

const (
	ActionAdd    DryRunAction = "ADD"
	ActionUpdate DryRunAction = "UPDATE"
	ActionDelete DryRunAction = "DELETE"
	ActionEqual  DryRunAction = "EQUAL"
	ActionNotice DryRunAction = "NOTICE"
)

type DryRunItem struct {
	Action  DryRunAction
	Path    string
	Size    int64
	Message string
}

type DryRunResult struct {
	Items            []DryRunItem
	ToAdd            int
	ToUpdate         int
	ToDelete         int
	Unchanged        int
	BytesTransferred int64
	Duration         time.Duration
	Err              error
}

type ActiveTransfer struct {
	Name       string  `json:"name"`
	Size       int64   `json:"size"`
	Bytes      int64   `json:"bytes"`
	Percentage int     `json:"percentage"`
	Speed      float64 `json:"speed"`
	SpeedAvg   float64 `json:"speedAvg"`
	ETA        int     `json:"eta"`
}

type StatsMsg struct {
	Bytes          int64            `json:"bytes"`
	TotalBytes     int64            `json:"totalBytes"`
	Speed          float64          `json:"speed"`
	SpeedAverage   float64          `json:"speedAverage"`
	Percentage     int              `json:"percentage"`
	ETA            int              `json:"eta"`
	Transfers      int              `json:"transfers"`
	TotalTransfers int              `json:"totalTransfers"`
	Errors         int              `json:"errors"`
	Checks         int              `json:"checks"`
	TotalChecks    int              `json:"totalChecks"`
	Transferring   []ActiveTransfer `json:"transferring"`
	RawLine        string           `json:"-"`
}

type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type TransferJob struct {
	ID          string
	ProfileID   string
	Name        string
	Source      string
	Destination string
	Operation   string
	IsDryRun    bool
	StartTime   time.Time
	EndTime     *time.Time
	Status      JobStatus
	LatestStats *StatsMsg
	Logs        []string
	ErrorMsg    string
	CancelFunc  context.CancelFunc
}

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func FormatSpeed(bytesPerSec float64) string {
	return fmt.Sprintf("%s/s", FormatBytes(int64(bytesPerSec)))
}

func FormatDuration(seconds int) string {
	if seconds <= 0 {
		return "--:--"
	}
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
	}
	return fmt.Sprintf("%02dm %02ds", m, s)
}
