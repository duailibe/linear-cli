package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUniquePathSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(base, []byte("a"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file-1.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file-1: %v", err)
	}

	got, err := uniquePath(base, false)
	if err != nil {
		t.Fatalf("uniquePath error: %v", err)
	}
	want := filepath.Join(dir, "file-2.txt")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestDownloadToFileCleansTempOnError(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("x"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		if hijacker, ok := w.(http.Hijacker); ok {
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close()
				return
			}
		}
	}))
	defer srv.Close()

	dest := filepath.Join(dir, "file.bin")
	err := downloadToFile(context.Background(), srv.URL, dest, "", 2*time.Second)
	if err == nil {
		t.Fatalf("expected error")
	}

	if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected destination not to exist, got %v", statErr)
	}

	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("read dir: %v", readErr)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".linear-attachment-") {
			t.Fatalf("expected no temp files, found %s", entry.Name())
		}
	}
}
