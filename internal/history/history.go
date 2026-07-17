// Package history persists a bounded log of executed requests so users can
// revisit and re-run past calls without re-typing them.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// MaxEntries caps how many requests are kept; oldest entries are dropped.
const MaxEntries = 50

// Entry records one executed request and its outcome.
type Entry struct {
	Method     string    `json:"method"`
	URL        string    `json:"url"`
	StatusCode int       `json:"status_code,omitempty"`
	Elapsed    string    `json:"elapsed,omitempty"`
	Err        string    `json:"error,omitempty"`
	At         time.Time `json:"at"`
}

// Store persists history as a single JSON file under BaseDir.
type Store struct {
	BaseDir string
}

func DefaultStore() *Store {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &Store{BaseDir: filepath.Join(home, ".curlmoon")}
}

func NewStore(baseDir string) *Store {
	return &Store{BaseDir: baseDir}
}

func (s *Store) path() string {
	return filepath.Join(s.BaseDir, "history.json")
}

// Load reads the saved history, newest first. Returns an empty slice if none exists.
func (s *Store) Load() ([]Entry, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) save(entries []Entry) error {
	if err := os.MkdirAll(s.BaseDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), data, 0o644)
}

// Add prepends a new entry and trims the log to MaxEntries, persisting the result.
func (s *Store) Add(e Entry) ([]Entry, error) {
	entries, err := s.Load()
	if err != nil {
		entries = nil
	}
	entries = append([]Entry{e}, entries...)
	if len(entries) > MaxEntries {
		entries = entries[:MaxEntries]
	}
	if err := s.save(entries); err != nil {
		return entries, err
	}
	return entries, nil
}
