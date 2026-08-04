// Package localfs is a storage.Store implementation that keeps uploaded
// bytes on local disk and metadata in memory. It's meant for local
// development and single-instance deployments; swap in an S3/GCS-backed
// storage.Store for anything that needs to survive restarts across
// multiple instances.
package localfs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"tuti-server/internal/storage"
)

// Store persists capture bytes under dir and tracks their metadata
// in memory.
type Store struct {
	dir string

	mu      sync.RWMutex
	objects map[string]storage.Object
}

// New returns a Store that writes files under dir, creating it if
// necessary.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("localfs: create dir: %w", err)
	}
	return &Store{
		dir:     dir,
		objects: make(map[string]storage.Object),
	}, nil
}

func (s *Store) Save(ctx context.Context, name string, contentType string, data []byte) (storage.Object, error) {
	id, err := newID()
	if err != nil {
		return storage.Object{}, fmt.Errorf("localfs: generate id: %w", err)
	}

	if err := os.WriteFile(s.path(id), data, 0o644); err != nil {
		return storage.Object{}, fmt.Errorf("localfs: write file: %w", err)
	}

	obj := storage.Object{
		ID:          id,
		Name:        name,
		ContentType: contentType,
		SizeBytes:   int64(len(data)),
		UploadedAt:  time.Now().UTC(),
	}

	s.mu.Lock()
	s.objects[id] = obj
	s.mu.Unlock()

	return obj, nil
}

func (s *Store) Get(ctx context.Context, id string) (storage.Object, []byte, error) {
	s.mu.RLock()
	obj, ok := s.objects[id]
	s.mu.RUnlock()
	if !ok {
		return storage.Object{}, nil, storage.ErrNotFound
	}

	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return storage.Object{}, nil, fmt.Errorf("localfs: read file: %w", err)
	}
	return obj, data, nil
}

func (s *Store) List(ctx context.Context) ([]storage.Object, error) {
	s.mu.RLock()
	objs := make([]storage.Object, 0, len(s.objects))
	for _, obj := range s.objects {
		objs = append(objs, obj)
	}
	s.mu.RUnlock()

	sort.Slice(objs, func(i, j int) bool {
		return objs[i].UploadedAt.After(objs[j].UploadedAt)
	})
	return objs, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id)
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "cap_" + hex.EncodeToString(b), nil
}
