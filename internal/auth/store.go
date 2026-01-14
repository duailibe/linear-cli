package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	authFileName = "auth.json"
)

type Store struct {
	Path string
}

type File struct {
	APIKey  string    `json:"api_key"`
	SavedAt time.Time `json:"saved_at"`
}

func DefaultStorePath() (string, error) {
	if base := os.Getenv("XDG_DATA_HOME"); base != "" {
		return filepath.Join(base, "linear", authFileName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".local", "share", "linear", authFileName), nil
}

func NewStore(path string) *Store {
	return &Store{Path: path}
}

func (s *Store) Load() (File, bool, error) {
	file, err := os.Open(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, false, nil
		}
		return File{}, false, fmt.Errorf("open auth file: %w", err)
	}
	defer file.Close()

	var data File
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return File{}, false, fmt.Errorf("decode auth file: %w", err)
	}

	if data.APIKey == "" {
		return File{}, false, nil
	}

	return data, true, nil
}

func (s *Store) Save(apiKey string, now time.Time) error {
	if apiKey == "" {
		return errors.New("api key is empty")
	}

	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create auth dir: %w", err)
	}

	tmp := s.Path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("write auth file: %w", err)
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(File{APIKey: apiKey, SavedAt: now}); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode auth file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close auth file: %w", err)
	}

	if err := os.Rename(tmp, s.Path); err != nil {
		return fmt.Errorf("replace auth file: %w", err)
	}

	return nil
}

func (s *Store) Delete() error {
	if err := os.Remove(s.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove auth file: %w", err)
	}
	return nil
}
