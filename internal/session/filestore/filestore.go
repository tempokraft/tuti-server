// Package filestore is a session.Store implementation that keeps each
// session record as one JSON file on local disk, mirroring
// internal/storage/localfs's role for captures. It's meant for local
// development and single-instance deployments: state survives a process
// restart, but there's no cross-process locking, so running multiple
// instances against the same dir would race.
package filestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tuti-server/internal/session"
)

// Store persists each (kind, id) record at dir/<kind>/<id>.json.
type Store struct {
	dir string
}

// New returns a Store rooted at dir, creating it if necessary.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("filestore: create dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(kind, id string, data []byte) error {
	dir := filepath.Join(s.dir, kind)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("filestore: create dir: %w", err)
	}
	if err := os.WriteFile(s.path(kind, id), data, 0o644); err != nil {
		return fmt.Errorf("filestore: write file: %w", err)
	}
	return nil
}

func (s *Store) Load(kind, id string) ([]byte, error) {
	data, err := os.ReadFile(s.path(kind, id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("filestore: read file: %w", err)
	}
	return data, nil
}

func (s *Store) List(kind string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, kind))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("filestore: read dir: %w", err)
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".json"))
	}
	return ids, nil
}

func (s *Store) path(kind, id string) string {
	return filepath.Join(s.dir, kind, id+".json")
}
