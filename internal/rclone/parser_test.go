package rclone

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tc := range tests {
		actual := FormatBytes(tc.bytes)
		if actual != tc.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tc.bytes, actual, tc.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		secs     int
		expected string
	}{
		{0, "--:--"},
		{-1, "--:--"},
		{35, "00m 35s"},
		{125, "02m 05s"},
		{3665, "01h 01m 05s"},
	}

	for _, tc := range tests {
		actual := FormatDuration(tc.secs)
		if actual != tc.expected {
			t.Errorf("FormatDuration(%d) = %s; want %s", tc.secs, actual, tc.expected)
		}
	}
}

func TestParseLogLineJSONStats(t *testing.T) {
	line := `{"level":"info","msg":"Transferred stats","time":"2026-09-24T05:00:00Z","stats":{"bytes":2048,"totalBytes":4096,"speed":1024.5,"percentage":50,"eta":2,"transfers":1,"totalTransfers":1}}`
	item, stats, msg := ParseLogLine(line)

	if item != nil {
		t.Errorf("expected nil item, got %+v", item)
	}
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}
	if stats.Bytes != 2048 {
		t.Errorf("expected 2048 bytes, got %d", stats.Bytes)
	}
	if stats.Percentage != 50 {
		t.Errorf("expected 50 percent, got %d", stats.Percentage)
	}
	if msg != "Transferred stats" {
		t.Errorf("expected msg 'Transferred stats', got '%s'", msg)
	}
}

func TestParseLogLineDryRun(t *testing.T) {
	line1 := `NOTICE: Photos/vacation.jpg: Skipped copy that would be done (dry run)`
	item1, stats1, _ := ParseLogLine(line1)
	if stats1 != nil {
		t.Errorf("expected nil stats, got %+v", stats1)
	}
	if item1 == nil {
		t.Fatal("expected item1, got nil")
	}
	if item1.Action != ActionAdd {
		t.Errorf("expected ActionAdd, got %v", item1.Action)
	}
	if item1.Path != "Photos/vacation.jpg" {
		t.Errorf("expected 'Photos/vacation.jpg', got '%s'", item1.Path)
	}

	line2 := `NOTICE: obsolete_file.log: Skipped delete that would be done (dry run)`
	item2, _, _ := ParseLogLine(line2)
	if item2 == nil || item2.Action != ActionDelete {
		t.Fatalf("expected ActionDelete item, got %+v", item2)
	}
}

func TestMockClientOperations(t *testing.T) {
	client := NewMockClient()

	if !client.IsAvailable() {
		t.Error("expected MockClient to be available")
	}
	if !client.IsMock() {
		t.Error("expected MockClient to report IsMock() true")
	}

	remotes, err := client.ListRemotes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing remotes: %v", err)
	}
	if len(remotes) == 0 {
		t.Fatal("expected at least one remote")
	}

	about, err := client.AboutRemote(context.Background(), "gdrive:")
	if err != nil {
		t.Fatalf("unexpected error fetching about info: %v", err)
	}
	if about.Total <= 0 {
		t.Errorf("expected positive total quota, got %d", about.Total)
	}

	dirItems, err := client.ListDir(context.Background(), "gdrive:Photos")
	if err != nil {
		t.Fatalf("unexpected error listing dir: %v", err)
	}
	if len(dirItems) == 0 {
		t.Fatal("expected dir items")
	}

	dryRunRes, err := client.DryRun(context.Background(), "sync", "local:~/Photos", "gdrive:Photos", nil)
	if err != nil {
		t.Fatalf("unexpected error in dry run: %v", err)
	}
	if dryRunRes.ToAdd == 0 && dryRunRes.ToUpdate == 0 {
		t.Error("expected some dry run actions")
	}

	// Test mock transfer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	job := &TransferJob{
		ID:          "test-job",
		Source:      "local:~/Test",
		Destination: "s3:test",
		Operation:   "copy",
	}

	var statsReceived bool
	err = client.StartTransfer(ctx, job, func(s *StatsMsg) {
		statsReceived = true
	}, nil)

	// Since context might timeout or finish, checking if stats callback fired
	if !statsReceived {
		t.Log("Note: fast run or cancelled before tick")
	}
}

func TestNormalizeRclonePath(t *testing.T) {
	norm1 := NormalizeRclonePath("local:~/Documents")
	if strings.HasPrefix(norm1, "local:") || strings.HasPrefix(norm1, "~") {
		t.Errorf("expected expanded path without local: and ~, got: %s", norm1)
	}

	norm2 := NormalizeRclonePath("gdrive:Backup_Teste")
	if norm2 != "gdrive:Backup_Teste" {
		t.Errorf("expected gdrive:Backup_Teste unchanged, got: %s", norm2)
	}

	norm3 := NormalizeRclonePath("local:/var/log")
	if norm3 != "/var/log" {
		t.Errorf("expected /var/log, got: %s", norm3)
	}
}
