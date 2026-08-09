package filestore

import (
	"errors"
	"testing"

	"tuti-server/internal/session"
)

func TestStore_LoadMissing(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := store.Load("snap", "does_not_exist"); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("Load: got %v, want session.ErrNotFound", err)
	}
}

func TestStore_ListEmptyKind(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// A kind that's never had anything saved under it shouldn't error —
	// its directory simply doesn't exist yet.
	ids, err := store.List("solve")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("List: got %v, want empty", ids)
	}
}

func TestStore_SaveLoadRoundTrip(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	want := []byte(`{"step":1}`)
	if err := store.Save("snap", "snap_abc", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load("snap", "snap_abc")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("Load = %q, want %q", got, want)
	}

	ids, err := store.List("snap")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ids) != 1 || ids[0] != "snap_abc" {
		t.Fatalf("List = %v, want [snap_abc]", ids)
	}
}

func TestStore_SaveOverwrites(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := store.Save("solve", "sess_1", []byte("first")); err != nil {
		t.Fatalf("Save (first): %v", err)
	}
	if err := store.Save("solve", "sess_1", []byte("second")); err != nil {
		t.Fatalf("Save (second): %v", err)
	}

	got, err := store.Load("solve", "sess_1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("Load = %q, want %q", got, "second")
	}

	ids, err := store.List("solve")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("List = %v, want exactly 1 entry", ids)
	}
}
