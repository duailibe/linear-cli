package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultStorePathXDG(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", temp)

	path, err := DefaultStorePath()
	if err != nil {
		t.Fatalf("DefaultStorePath() error: %v", err)
	}

	expected := filepath.Join(temp, "linear", "auth.json")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestStoreSaveLoadDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	store := NewStore(path)
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	if err := store.Save("test-key", now); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok true")
	}
	if data.APIKey != "test-key" {
		t.Fatalf("expected api key saved")
	}

	if deleteErr := store.Delete(); deleteErr != nil {
		t.Fatalf("Delete() error: %v", deleteErr)
	}
	_, ok, err = store.Load()
	if err != nil {
		t.Fatalf("Load() after delete error: %v", err)
	}
	if ok {
		t.Fatalf("expected no auth after delete")
	}
}

func TestStoreFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	store := NewStore(path)

	if err := store.Save("test-key", time.Now()); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected permissions 0600, got %v", info.Mode().Perm())
	}
}
