package service

import (
	"bytes"
	"context"
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
	// ErrStateMemoryLimitReached indicates that stored payloads reached their global byte budget.
	ErrStateMemoryLimitReached = errors.New("state memory limit reached")
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
	maxTotalBytes   int64
	totalBytes      int64
	now             func() time.Time
}

// NewStateStore creates a store with fixed entry and payload limits.
//
// All limits must be positive. maxTotalBytes limits owned payload bytes across
// all entries; snapshots returned to callers are not included.
func NewStateStore(maxEntries int, maxPayloadBytes, maxTotalBytes int64) *StateStore {
	if maxEntries <= 0 {
		panic("state store maxEntries must be positive")
	}
	if maxPayloadBytes <= 0 {
		panic("state store maxPayloadBytes must be positive")
	}
	if maxTotalBytes <= 0 {
		panic("state store maxTotalBytes must be positive")
	}
	return &StateStore{
		entries:         make(map[string]stateEntry),
		maxEntries:      maxEntries,
		maxPayloadBytes: maxPayloadBytes,
		maxTotalBytes:   maxTotalBytes,
		now:             time.Now,
	}
}

// RunCleanup removes expired entries at interval until ctx is canceled.
// The method blocks and is intended to run in one application-owned goroutine.
func (s *StateStore) RunCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		panic("state cleanup interval must be positive")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.mu.Lock()
			s.cleanupLocked(now)
			s.mu.Unlock()
		}
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

	existing, exists := s.entries[key]
	if !exists && len(s.entries) >= s.maxEntries {
		return StoredState{}, false, ErrStateCapacityReached
	}
	existingBytes := int64(len(existing.payload))
	newTotalBytes := s.totalBytes - existingBytes + int64(len(payload))
	if newTotalBytes > s.maxTotalBytes {
		return StoredState{}, false, ErrStateMemoryLimitReached
	}

	ownedPayload := bytes.Clone(payload)
	s.entries[key] = stateEntry{payload: ownedPayload, expiresAt: expiresAt}
	s.totalBytes = newTotalBytes
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

	entry, ok := s.entries[key]
	if !ok {
		return false
	}
	delete(s.entries, key)
	s.totalBytes -= int64(len(entry.payload))
	return true
}

func (s *StateStore) cleanupLocked(now time.Time) {
	for key, entry := range s.entries {
		if !entry.expiresAt.After(now) {
			delete(s.entries, key)
			s.totalBytes -= int64(len(entry.payload))
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
