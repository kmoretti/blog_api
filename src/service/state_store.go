package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

var (
	// ErrInvalidStatePayload indicates that a payload is not a JSON object.
	ErrInvalidStatePayload = errors.New("state payload must be a JSON object")
	// ErrStatePayloadTooLarge indicates that a payload exceeds the configured limit.
	ErrStatePayloadTooLarge = errors.New("state payload is too large")
	// ErrStateCapacityReached indicates that no more distinct states can be stored.
	ErrStateCapacityReached = errors.New("state capacity reached")
)

type stateEntry struct {
	payload   json.RawMessage
	expiresAt time.Time
}

// StoredState is an immutable snapshot returned by StateStore.
type StoredState struct {
	Payload   json.RawMessage
	ExpiresAt time.Time
}

// StateStore owns bounded, expiring JSON states in process memory.
//
// State values disappear when the process exits. Put replaces a value in full;
// the store never interprets or mutates fields inside the JSON object.
type StateStore struct {
	mu              sync.Mutex
	entries         map[string]stateEntry
	maxEntries      int
	maxPayloadBytes int64
	now             func() time.Time
}

// NewStateStore creates a store with fixed entry and payload limits.
//
// maxEntries and maxPayloadBytes must both be positive.
func NewStateStore(maxEntries int, maxPayloadBytes int64) *StateStore {
	if maxEntries <= 0 {
		panic("state store maxEntries must be positive")
	}
	if maxPayloadBytes <= 0 {
		panic("state store maxPayloadBytes must be positive")
	}
	return &StateStore{
		entries:         make(map[string]stateEntry),
		maxEntries:      maxEntries,
		maxPayloadBytes: maxPayloadBytes,
		now:             time.Now,
	}
}

// Put stores a JSON object until the supplied expiration time.
//
// The payload is copied before Put returns. created reports whether key was new.
func (s *StateStore) Put(key string, payload []byte, expiresAt time.Time) (state StoredState, created bool, err error) {
	if int64(len(payload)) > s.maxPayloadBytes {
		return StoredState{}, false, ErrStatePayloadTooLarge
	}
	if !isJSONObject(payload) {
		return StoredState{}, false, ErrInvalidStatePayload
	}

	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)

	_, exists := s.entries[key]
	if !exists && len(s.entries) >= s.maxEntries {
		return StoredState{}, false, ErrStateCapacityReached
	}

	ownedPayload := bytes.Clone(payload)
	s.entries[key] = stateEntry{payload: ownedPayload, expiresAt: expiresAt}
	return snapshot(s.entries[key]), !exists, nil
}

// Get returns a copied snapshot for key and lazily removes expired states.
func (s *StateStore) Get(key string) (StoredState, bool) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)

	entry, ok := s.entries[key]
	if !ok {
		return StoredState{}, false
	}
	return snapshot(entry), true
}

// Delete removes key and reports whether it existed and had not expired.
func (s *StateStore) Delete(key string) bool {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)

	if _, ok := s.entries[key]; !ok {
		return false
	}
	delete(s.entries, key)
	return true
}

func (s *StateStore) cleanupLocked(now time.Time) {
	for key, entry := range s.entries {
		if !entry.expiresAt.After(now) {
			delete(s.entries, key)
		}
	}
}

func snapshot(entry stateEntry) StoredState {
	return StoredState{
		Payload:   bytes.Clone(entry.payload),
		ExpiresAt: entry.expiresAt,
	}
}

func isJSONObject(payload []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var object map[string]json.RawMessage
	if err := decoder.Decode(&object); err != nil || object == nil {
		return false
	}
	var trailing interface{}
	return errors.Is(decoder.Decode(&trailing), io.EOF)
}
