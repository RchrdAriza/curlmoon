package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Session captures the request editor's in-progress state so it survives
// restarts even if the user never explicitly saved it into a collection.
type Session struct {
	Method    string   `json:"method"`
	URL       string   `json:"url"`
	Headers   []KeyVal `json:"headers,omitempty"`
	Params    []KeyVal `json:"params,omitempty"`
	BodyType  string   `json:"body_type,omitempty"`
	Body      string   `json:"body,omitempty"`
	AuthType  string   `json:"auth_type,omitempty"`
	AuthText  string   `json:"auth_text,omitempty"`
	Scripts   string   `json:"scripts,omitempty"`
	ActiveTab int      `json:"active_tab"`
}

func (s *Store) sessionPath() string {
	return filepath.Join(s.BaseDir, "session.json")
}

// SaveSession writes the current session to disk, overwriting any previous one.
func (s *Store) SaveSession(sess *Session) error {
	if err := os.MkdirAll(s.BaseDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.sessionPath(), data, 0o644)
}

// LoadSession reads the last saved session, returning (nil, nil) if none exists.
func (s *Store) LoadSession() (*Session, error) {
	data, err := os.ReadFile(s.sessionPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}
