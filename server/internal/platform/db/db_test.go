package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAppliesPragmasAndCreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "test.db")
	gdb, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close(gdb) })

	var mode string
	if err := gdb.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}

	var timeout int
	if err := gdb.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil {
		t.Fatal(err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("db file not created: %v", err)
	}
}
