// Package storage defines the abstraction over where uploaded captures
// (screenshots of a student's math problem or work) are persisted, so a
// local-disk implementation can be swapped for cloud object storage (S3,
// GCS, ...) without touching the HTTP layer.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by Get when id has no matching object.
var ErrNotFound = errors.New("storage: object not found")

// Object is the metadata for a stored capture.
type Object struct {
	ID          string
	Name        string
	ContentType string
	SizeBytes   int64
	UploadedAt  time.Time
}

// Store persists and retrieves uploaded captures.
type Store interface {
	// Save stores data under a newly assigned ID and returns its metadata.
	Save(ctx context.Context, name string, contentType string, data []byte) (Object, error)

	// Get returns the metadata and bytes for a previously saved object.
	// Returns ErrNotFound if id does not exist.
	Get(ctx context.Context, id string) (Object, []byte, error)

	// List returns all stored objects' metadata, most recently uploaded
	// first.
	List(ctx context.Context) ([]Object, error)
}
