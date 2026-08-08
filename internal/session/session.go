// Package session holds the server-side state for tuti-server's two
// stateful flows: the step-driven Snap & Solve flow
// (InitializeSnapAndSolve -> SubmitSnap -> SubmitSnapResponse) and the
// looser CreateSession/AnalyzeAssets pairing. Both keep an in-memory index
// for fast lookups, write-through to a Store for durability, and rehydrate
// from it on startup — so state survives a restart. See Store's doc
// comment for the persistence contract, and internal/session/filestore for
// the local-disk implementation.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNotFound is returned when a session id doesn't exist — either from
// SnapStore/SolveStore lookups, or from a Store implementation's Load.
var ErrNotFound = errors.New("session: not found")

// ErrWrongStep is returned when a Snap & Solve call is made out of order
// (e.g. submitting a response before a snap was ever uploaded).
var ErrWrongStep = errors.New("session: wrong step for this call")

// Store durably persists session records so SnapStore/SolveStore survive a
// process restart. A record is opaque, caller-marshaled bytes — each
// session type owns its own JSON shape — addressed by (kind, id). kind
// namespaces the two flows sharing this package ("snap" vs "solve"); their
// id spaces are already disjoint via the "snap_"/"sess_" prefixes in
// newID, but kind keeps a file-based implementation's on-disk layout
// legible either way, and leaves room for a future kind without a
// migration.
type Store interface {
	// Save durably writes data for (kind, id), creating or overwriting it.
	Save(kind, id string, data []byte) error
	// Load reads back data previously written via Save. Returns
	// ErrNotFound if nothing has been saved under (kind, id).
	Load(kind, id string) ([]byte, error)
	// List returns the ids of every record saved under kind.
	List(kind string) ([]string, error)
}

func newID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

// ── Snap & Solve flow ───────────────────────────────────────────────────────

type snapStep int

const (
	stepAwaitSnap snapStep = iota
	stepAwaitResponse
	stepDone
)

const snapKind = "snap"

// snapSession's fields are exported (unlike the rest of this unexported
// type) because encoding/json only sees exported fields — this is the
// on-disk JSON shape, not just an in-memory struct.
type snapSession struct {
	Step      snapStep  `json:"step"`
	CaptureID string    `json:"captureId"`
	CreatedAt time.Time `json:"createdAt"`
}

// SnapStore tracks in-progress Snap & Solve sessions.
type SnapStore struct {
	store Store

	mu       sync.Mutex
	sessions map[string]*snapSession
}

// NewSnapStore returns a SnapStore backed by store, rehydrated from
// whatever it already holds (empty on a first run).
func NewSnapStore(store Store) (*SnapStore, error) {
	s := &SnapStore{store: store, sessions: make(map[string]*snapSession)}

	ids, err := store.List(snapKind)
	if err != nil {
		return nil, fmt.Errorf("session: list snap sessions: %w", err)
	}
	for _, id := range ids {
		data, err := store.Load(snapKind, id)
		if err != nil {
			return nil, fmt.Errorf("session: load snap session %s: %w", id, err)
		}
		var sess snapSession
		if err := json.Unmarshal(data, &sess); err != nil {
			return nil, fmt.Errorf("session: decode snap session %s: %w", id, err)
		}
		s.sessions[id] = &sess
	}
	return s, nil
}

func (s *SnapStore) persist(id string, sess *snapSession) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session: encode snap session %s: %w", id, err)
	}
	if err := s.store.Save(snapKind, id, data); err != nil {
		return fmt.Errorf("session: save snap session %s: %w", id, err)
	}
	return nil
}

// Init starts a new Snap & Solve session, awaiting a photo upload.
func (s *SnapStore) Init() (string, error) {
	id, err := newID("snap_")
	if err != nil {
		return "", err
	}

	sess := &snapSession{Step: stepAwaitSnap, CreatedAt: time.Now()}
	if err := s.persist(id, sess); err != nil {
		return "", err
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return id, nil
}

// SubmitSnap records the uploaded capture id for sessionID and advances it
// to await the student's chosen response option.
func (s *SnapStore) SubmitSnap(sessionID, captureID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrNotFound
	}
	if sess.Step != stepAwaitSnap {
		return ErrWrongStep
	}

	// Build the next state and persist it before committing the mutation
	// in memory, so a failed write leaves the session at its last
	// durably-saved step instead of one the caller was told succeeded.
	next := *sess
	next.CaptureID = captureID
	next.Step = stepAwaitResponse
	if err := s.persist(sessionID, &next); err != nil {
		return err
	}
	*sess = next
	return nil
}

// SubmitResponse marks sessionID done and returns the capture id captured
// earlier via SubmitSnap.
func (s *SnapStore) SubmitResponse(sessionID string) (captureID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return "", ErrNotFound
	}
	if sess.Step != stepAwaitResponse {
		return "", ErrWrongStep
	}

	next := *sess
	next.Step = stepDone
	if err := s.persist(sessionID, &next); err != nil {
		return "", err
	}
	*sess = next
	return sess.CaptureID, nil
}

// ── Solve session (CreateSession / AnalyzeAssets) ──────────────────────────

const solveKind = "solve"

// SolveSession is a lightweight handle created by CreateSession and
// referenced by subsequent AnalyzeAssets calls.
type SolveSession struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

// SolveStore tracks known solve sessions (just enough to validate that an
// AnalyzeAssets call references a session that was actually created —
// asset ids themselves are passed explicitly on each AnalyzeAssets call
// rather than accumulated server-side).
type SolveStore struct {
	store Store

	mu       sync.Mutex
	sessions map[string]SolveSession
}

// NewSolveStore returns a SolveStore backed by store, rehydrated from
// whatever it already holds (empty on a first run).
func NewSolveStore(store Store) (*SolveStore, error) {
	s := &SolveStore{store: store, sessions: make(map[string]SolveSession)}

	ids, err := store.List(solveKind)
	if err != nil {
		return nil, fmt.Errorf("session: list solve sessions: %w", err)
	}
	for _, id := range ids {
		data, err := store.Load(solveKind, id)
		if err != nil {
			return nil, fmt.Errorf("session: load solve session %s: %w", id, err)
		}
		var sess SolveSession
		if err := json.Unmarshal(data, &sess); err != nil {
			return nil, fmt.Errorf("session: decode solve session %s: %w", id, err)
		}
		s.sessions[id] = sess
	}
	return s, nil
}

// Create starts a new solve session.
func (s *SolveStore) Create() (SolveSession, error) {
	id, err := newID("sess_")
	if err != nil {
		return SolveSession{}, err
	}
	sess := SolveSession{ID: id, CreatedAt: time.Now()}

	data, err := json.Marshal(sess)
	if err != nil {
		return SolveSession{}, fmt.Errorf("session: encode solve session %s: %w", id, err)
	}
	if err := s.store.Save(solveKind, id, data); err != nil {
		return SolveSession{}, fmt.Errorf("session: save solve session %s: %w", id, err)
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return sess, nil
}

// Exists reports whether sessionID was created via Create.
func (s *SolveStore) Exists(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[sessionID]
	return ok
}
