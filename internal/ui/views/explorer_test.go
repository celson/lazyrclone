package views

import (
	"testing"
)

func TestBuildRemotePath(t *testing.T) {
	tests := []struct {
		remote   string
		dir      string
		name     string
		expected string
	}{
		{"local:", ".", "file.txt", "file.txt"},
		{"local:", "", "file.txt", "file.txt"},
		{"local:", "Documents", "file.txt", "Documents/file.txt"},
		{"local:", "Documents/Work", "", "Documents/Work"},
		{"local:", "", "", "."},
		{"gdrive:", "", "Photos", "gdrive:Photos"},
		{"gdrive:", "Backup", "file.txt", "gdrive:Backup/file.txt"},
		{"gdrive:", "Backup/2026", "", "gdrive:Backup/2026"},
		{"s3:bucket", "logs", "app.log", "s3:bucket/logs/app.log"},
	}

	for _, tc := range tests {
		got := BuildRemotePath(tc.remote, tc.dir, tc.name)
		if got != tc.expected {
			t.Errorf("BuildRemotePath(%q, %q, %q) = %q, expected %q", tc.remote, tc.dir, tc.name, got, tc.expected)
		}
	}
}
