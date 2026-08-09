package session_test

import (
	"errors"
	"testing"

	"tuti-server/internal/session"
	"tuti-server/internal/session/filestore"
)

// newStore returns a filestore.Store rooted at a fresh temp dir. Using the
// real file-backed Store (rather than an in-memory fake) is deliberate:
// what this package needs proven is that state actually survives a
// restart, which only an on-disk backend can demonstrate.
func newStore(t *testing.T) session.Store {
	t.Helper()
	store, err := filestore.New(t.TempDir())
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	return store
}

func TestSnapStore_SubmitSnap_UnknownSession(t *testing.T) {
	store := newStore(t)
	snaps, err := session.NewSnapStore(store)
	if err != nil {
		t.Fatalf("NewSnapStore: %v", err)
	}

	if err := snaps.SubmitSnap("snap_does_not_exist", "cap_x"); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("SubmitSnap: got %v, want ErrNotFound", err)
	}
}

func TestSnapStore_WrongStepOrder(t *testing.T) {
	store := newStore(t)
	snaps, err := session.NewSnapStore(store)
	if err != nil {
		t.Fatalf("NewSnapStore: %v", err)
	}

	id, err := snaps.Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	// SubmitResponse before SubmitSnap is out of order.
	if _, err := snaps.SubmitResponse(id); !errors.Is(err, session.ErrWrongStep) {
		t.Fatalf("SubmitResponse: got %v, want ErrWrongStep", err)
	}
}

func TestSnapStore_PersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	store, err := filestore.New(dir)
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}

	snaps, err := session.NewSnapStore(store)
	if err != nil {
		t.Fatalf("NewSnapStore: %v", err)
	}

	id, err := snaps.Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := snaps.SubmitSnap(id, "cap_abc123"); err != nil {
		t.Fatalf("SubmitSnap: %v", err)
	}

	// Simulate a process restart: a brand new Store and SnapStore pointed
	// at the same directory, with nothing carried over in memory.
	restartedStore, err := filestore.New(dir)
	if err != nil {
		t.Fatalf("filestore.New (restart): %v", err)
	}
	restarted, err := session.NewSnapStore(restartedStore)
	if err != nil {
		t.Fatalf("NewSnapStore (restart): %v", err)
	}

	captureID, err := restarted.SubmitResponse(id)
	if err != nil {
		t.Fatalf("SubmitResponse after restart: %v", err)
	}
	if captureID != "cap_abc123" {
		t.Fatalf("captureID = %q, want %q", captureID, "cap_abc123")
	}

	// And the step advanced past stepAwaitResponse for real, not just in
	// the restarted process's memory: a second SubmitResponse must fail.
	if _, err := restarted.SubmitResponse(id); !errors.Is(err, session.ErrWrongStep) {
		t.Fatalf("second SubmitResponse: got %v, want ErrWrongStep", err)
	}
}

func TestSolveStore_ExistsUnknownSession(t *testing.T) {
	store := newStore(t)
	solves, err := session.NewSolveStore(store)
	if err != nil {
		t.Fatalf("NewSolveStore: %v", err)
	}

	if solves.Exists("sess_does_not_exist") {
		t.Fatal("Exists: want false for a session that was never created")
	}
}

func TestSolveStore_PersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	store, err := filestore.New(dir)
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}

	solves, err := session.NewSolveStore(store)
	if err != nil {
		t.Fatalf("NewSolveStore: %v", err)
	}
	sess, err := solves.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	restartedStore, err := filestore.New(dir)
	if err != nil {
		t.Fatalf("filestore.New (restart): %v", err)
	}
	restarted, err := session.NewSolveStore(restartedStore)
	if err != nil {
		t.Fatalf("NewSolveStore (restart): %v", err)
	}

	if !restarted.Exists(sess.ID) {
		t.Fatalf("Exists(%q) = false after restart, want true", sess.ID)
	}
}
