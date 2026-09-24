package rclone

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) IsAvailable() bool {
	return true
}

func (m *MockClient) IsMock() bool {
	return true
}

func (m *MockClient) Version() (string, error) {
	return "rclone v1.68.0 (simulated demo mode)", nil
}

func (m *MockClient) ListRemotes(ctx context.Context) ([]RemoteInfo, error) {
	return []RemoteInfo{
		{
			Name: "gdrive:",
			Type: "drive",
			Details: map[string]string{
				"scope": "drive",
				"user":  "user@gmail.com",
			},
		},
		{
			Name: "s3:",
			Type: "s3",
			Details: map[string]string{
				"provider": "AWS",
				"region":   "us-east-1",
				"endpoint": "s3.amazonaws.com",
			},
		},
		{
			Name: "onedrive:",
			Type: "onedrive",
			Details: map[string]string{
				"drive_type": "personal",
			},
		},
		{
			Name: "dropbox:",
			Type: "dropbox",
			Details: map[string]string{
				"client_id": "app_dropbox_cloud",
			},
		},
		{
			Name: "local:",
			Type: "local",
			Details: map[string]string{
				"path": "/home/user",
			},
		},
	}, nil
}

func (m *MockClient) AboutRemote(ctx context.Context, remote string) (*AboutInfo, error) {
	switch {
	case strings.Contains(remote, "gdrive"):
		return &AboutInfo{
			Total:   100 * 1024 * 1024 * 1024, // 100 GB
			Used:    62 * 1024 * 1024 * 1024,  // 62 GB
			Free:    38 * 1024 * 1024 * 1024,  // 38 GB
			Objects: 14520,
		}, nil
	case strings.Contains(remote, "s3"):
		return &AboutInfo{
			Total:   500 * 1024 * 1024 * 1024,
			Used:    140 * 1024 * 1024 * 1024,
			Free:    360 * 1024 * 1024 * 1024,
			Objects: 52100,
		}, nil
	default:
		return &AboutInfo{
			Total:   15 * 1024 * 1024 * 1024,
			Used:    4 * 1024 * 1024 * 1024,
			Free:    11 * 1024 * 1024 * 1024,
			Objects: 3120,
		}, nil
	}
}

func (m *MockClient) ListDir(ctx context.Context, remotePath string) ([]FileItem, error) {
	now := time.Now()
	// If path is local, try local real directory first
	if strings.HasPrefix(remotePath, "local:") || strings.HasPrefix(remotePath, "/") {
		clean := strings.TrimPrefix(remotePath, "local:")
		if clean == "" || clean == "." {
			clean = "."
		}
		if items, err := listLocalDir(clean); err == nil && len(items) > 0 {
			return items, nil
		}
	}

	// Generate realistic remote items
	return []FileItem{
		{Name: "Documents", Path: "Documents", IsDir: true, ModTime: now.Add(-48 * time.Hour)},
		{Name: "Photos", Path: "Photos", IsDir: true, ModTime: now.Add(-12 * time.Hour)},
		{Name: "Backups", Path: "Backups", IsDir: true, ModTime: now.Add(-5 * 24 * time.Hour)},
		{Name: "Projects", Path: "Projects", IsDir: true, ModTime: now.Add(-2 * time.Hour)},
		{Name: "database_dump_2026.sql.gz", Path: "database_dump_2026.sql.gz", Size: 412 * 1024 * 1024, IsDir: false, ModTime: now.Add(-1 * time.Hour), MimeType: "application/gzip"},
		{Name: "quarterly_financials.xlsx", Path: "quarterly_financials.xlsx", Size: 4 * 1024 * 1024, IsDir: false, ModTime: now.Add(-24 * time.Hour), MimeType: "application/vnd.ms-excel"},
		{Name: "system_architecture.png", Path: "system_architecture.png", Size: 12 * 1024 * 1024, IsDir: false, ModTime: now.Add(-6 * time.Hour), MimeType: "image/png"},
		{Name: "dataset_raw.parquet", Path: "dataset_raw.parquet", Size: 1420 * 1024 * 1024, IsDir: false, ModTime: now.Add(-72 * time.Hour), MimeType: "application/octet-stream"},
		{Name: "notes_and_ideas.md", Path: "notes_and_ideas.md", Size: 18 * 1024, IsDir: false, ModTime: now.Add(-15 * time.Minute), MimeType: "text/markdown"},
	}, nil
}

func (m *MockClient) CreateDir(ctx context.Context, remotePath string) error {
	return nil
}

func (m *MockClient) Delete(ctx context.Context, remotePath string, isDir bool) error {
	return nil
}

func (m *MockClient) DryRun(ctx context.Context, op string, src string, dest string, flags []string) (*DryRunResult, error) {
	time.Sleep(300 * time.Millisecond) // simulate check time

	items := []DryRunItem{
		{Action: ActionAdd, Path: "Documents/Proposal_2026.pdf", Size: 3 * 1024 * 1024, Message: "Will be copied to destination"},
		{Action: ActionAdd, Path: "Photos/IMG_9912.raw", Size: 45 * 1024 * 1024, Message: "Will be copied to destination"},
		{Action: ActionAdd, Path: "Photos/IMG_9913.raw", Size: 48 * 1024 * 1024, Message: "Will be copied to destination"},
		{Action: ActionUpdate, Path: "database_dump_2026.sql.gz", Size: 412 * 1024 * 1024, Message: "Source is newer (size changed)"},
		{Action: ActionUpdate, Path: "quarterly_financials.xlsx", Size: 4 * 1024 * 1024, Message: "Source modified time is newer"},
		{Action: ActionDelete, Path: "old_temp_cache.bin", Size: 128 * 1024 * 1024, Message: "File deleted on source, would be purged on dest"},
		{Action: ActionDelete, Path: "obsolete_backup_v1.tar", Size: 250 * 1024 * 1024, Message: "File deleted on source, would be purged on dest"},
		{Action: ActionEqual, Path: "system_architecture.png", Size: 12 * 1024 * 1024, Message: "Checksums match, skipped"},
		{Action: ActionEqual, Path: "notes_and_ideas.md", Size: 18 * 1024, Message: "Identical size and time, skipped"},
	}

	res := &DryRunResult{
		Items:            items,
		ToAdd:            3,
		ToUpdate:         2,
		ToDelete:         2,
		Unchanged:        2,
		BytesTransferred: 512 * 1024 * 1024,
		Duration:         312 * time.Millisecond,
	}

	return res, nil
}

func (m *MockClient) StartTransfer(ctx context.Context, job *TransferJob, onStats func(*StatsMsg), onLog func(string)) error {
	totalBytes := int64(620 * 1024 * 1024)
	bytesTransferred := int64(0)
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	if onLog != nil {
		onLog(fmt.Sprintf("Starting %s: %s -> %s", job.Operation, job.Source, job.Destination))
		onLog("Connected to remote endpoint, verifying file indexes...")
	}

	mockFiles := []struct {
		name string
		size int64
	}{
		{"Photos/IMG_9912.raw", 45 * 1024 * 1024},
		{"Photos/IMG_9913.raw", 48 * 1024 * 1024},
		{"database_dump_2026.sql.gz", 412 * 1024 * 1024},
		{"Documents/Proposal_2026.pdf", 3 * 1024 * 1024},
		{"quarterly_financials.xlsx", 4 * 1024 * 1024},
	}

	for {
		select {
		case <-ctx.Done():
			if onLog != nil {
				onLog("Transfer aborted by user.")
			}
			return ctx.Err()
		case <-ticker.C:
			step := int64(30*1024*1024 + rand.Intn(15*1024*1024))
			bytesTransferred += step
			if bytesTransferred > totalBytes {
				bytesTransferred = totalBytes
			}

			pct := int((float64(bytesTransferred) / float64(totalBytes)) * 100)
			speed := 35.5 * 1024 * 1024 // ~35 MB/s
			remBytes := totalBytes - bytesTransferred
			eta := 0
			if speed > 0 {
				eta = int(float64(remBytes) / speed)
			}

			active := make([]ActiveTransfer, 0)
			for i, f := range mockFiles {
				filePct := (pct + i*15) % 100
				if pct >= 100 {
					filePct = 100
				}
				active = append(active, ActiveTransfer{
					Name:       f.name,
					Size:       f.size,
					Bytes:      int64(float64(f.size) * float64(filePct) / 100.0),
					Percentage: filePct,
					Speed:      7.2 * 1024 * 1024,
					ETA:        eta,
				})
			}

			stats := &StatsMsg{
				Bytes:          bytesTransferred,
				TotalBytes:     totalBytes,
				Speed:          speed,
				SpeedAverage:   speed,
				Percentage:     pct,
				ETA:            eta,
				Transfers:      len(mockFiles),
				TotalTransfers: len(mockFiles),
				Transferring:   active,
			}

			if onStats != nil {
				onStats(stats)
			}

			if onLog != nil && rand.Float32() < 0.3 {
				onLog(fmt.Sprintf("Transferred chunk %s at %s", FormatBytes(bytesTransferred), FormatSpeed(speed)))
			}

			if bytesTransferred >= totalBytes {
				if onLog != nil {
					onLog(fmt.Sprintf("Success: Transferred %s in 6.4s (38.2 MB/s). 0 errors.", FormatBytes(totalBytes)))
				}
				return nil
			}
		}
	}
}

// GetClient automatically detects if rclone is available, or returns mock client
func GetClient(binaryPath string, forceMock bool) RcloneClient {
	if forceMock {
		return NewMockClient()
	}
	real := NewRealClient(binaryPath)
	if real.IsAvailable() {
		return real
	}
	return NewMockClient()
}
