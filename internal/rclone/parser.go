package rclone

import (
	"encoding/json"
	"strings"
	"time"
)

type rawRcloneLog struct {
	Level   string    `json:"level"`
	Msg     string    `json:"msg"`
	Object  string    `json:"object"`
	Source  string    `json:"source"`
	Skipped string    `json:"skipped"`
	Size    int64     `json:"size"`
	Time    time.Time `json:"time"`
	Stats   *StatsMsg `json:"stats,omitempty"`
}

// ParseLogLine parses a JSON log line from rclone
func ParseLogLine(line string) (*DryRunItem, *StatsMsg, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil, ""
	}

	var raw rawRcloneLog
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		// Not JSON, return as plain text line
		return parsePlainTextDryRun(line), nil, line
	}

	// Check if this log has embedded stats
	if raw.Stats != nil {
		raw.Stats.RawLine = line
		return nil, raw.Stats, raw.Msg
	}

	// Ignore purely directory time setting notifications in dry-run
	if raw.Skipped == "set directory modification time" {
		return nil, nil, raw.Msg
	}

	// Check for dry-run actions in message or skipped field
	msgLower := strings.ToLower(raw.Msg)
	obj := raw.Object
	if obj == "" {
		obj = raw.Source
	}

	if obj != "" {
		if raw.Skipped == "copy" || strings.Contains(msgLower, "copy") || strings.Contains(msgLower, "new") || strings.Contains(msgLower, "created") {
			return &DryRunItem{
				Action:  ActionAdd,
				Path:    obj,
				Size:    raw.Size,
				Message: raw.Msg,
			}, nil, raw.Msg
		}
		if raw.Skipped == "delete" || strings.Contains(msgLower, "delete") || strings.Contains(msgLower, "removed") {
			return &DryRunItem{
				Action:  ActionDelete,
				Path:    obj,
				Size:    raw.Size,
				Message: raw.Msg,
			}, nil, raw.Msg
		}
		if strings.Contains(msgLower, "update") || strings.Contains(msgLower, "modify") || strings.Contains(msgLower, "differ") {
			return &DryRunItem{
				Action:  ActionUpdate,
				Path:    obj,
				Size:    raw.Size,
				Message: raw.Msg,
			}, nil, raw.Msg
		}
	}

	return nil, nil, raw.Msg
}

// parsePlainTextDryRun parses lines like "NOTICE: file.txt: Skipped copy that would be done (dry run)"
func parsePlainTextDryRun(line string) *DryRunItem {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "dry run") && !strings.Contains(lower, "notice") {
		return nil
	}

	parts := strings.SplitN(line, ":", 3)
	if len(parts) >= 3 {
		path := strings.TrimSpace(parts[1])
		msg := strings.TrimSpace(parts[2])

		switch {
		case strings.Contains(lower, "copy"):
			return &DryRunItem{Action: ActionAdd, Path: path, Message: msg}
		case strings.Contains(lower, "delete"):
			return &DryRunItem{Action: ActionDelete, Path: path, Message: msg}
		case strings.Contains(lower, "update") || strings.Contains(lower, "modify"):
			return &DryRunItem{Action: ActionUpdate, Path: path, Message: msg}
		default:
			return &DryRunItem{Action: ActionNotice, Path: path, Message: msg}
		}
	}

	return nil
}
