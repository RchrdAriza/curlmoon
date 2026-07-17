package collection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Store persists collections and session state as JSON files under BaseDir
// (defaults to ~/.curlmoon), mirroring how Postman keeps local collections.
type Store struct {
	BaseDir string
}

// DefaultStore points at ~/.curlmoon, falling back to the current directory
// if the home directory can't be resolved.
func DefaultStore() *Store {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &Store{BaseDir: filepath.Join(home, ".curlmoon")}
}

// NewStore points at an explicit base directory (mainly for tests).
func NewStore(baseDir string) *Store {
	return &Store{BaseDir: baseDir}
}

func (s *Store) collectionsDir() string {
	return filepath.Join(s.BaseDir, "collections")
}

func (s *Store) ensureDir() error {
	return os.MkdirAll(s.collectionsDir(), 0o755)
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "collection"
	}
	return s
}

func (s *Store) path(name string) string {
	return filepath.Join(s.collectionsDir(), slugify(name)+".json")
}

// List returns the names of all saved collections, sorted alphabetically.
func (s *Store) List() ([]string, error) {
	cols, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Info.Name
	}
	return names, nil
}

// LoadAll reads every collection file, skipping any that fail to parse.
func (s *Store) LoadAll() ([]*Collection, error) {
	if err := s.ensureDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.collectionsDir())
	if err != nil {
		return nil, err
	}
	var cols []*Collection
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		c, err := s.loadFile(filepath.Join(s.collectionsDir(), e.Name()))
		if err != nil {
			continue
		}
		cols = append(cols, c)
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].Info.Name < cols[j].Info.Name })
	return cols, nil
}

func (s *Store) loadFile(path string) (*Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Collection
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Load reads a single collection by name.
func (s *Store) Load(name string) (*Collection, error) {
	return s.loadFile(s.path(name))
}

// Save writes the collection to disk, keyed by its current Info.Name.
func (s *Store) Save(c *Collection) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	if c.Info.Schema == "" {
		c.Info.Schema = schemaURL
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(c.Info.Name), data, 0o644)
}

// Create makes a new empty collection and persists it.
func (s *Store) Create(name string) (*Collection, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}
	if _, err := os.Stat(s.path(name)); err == nil {
		return nil, fmt.Errorf("collection %q already exists", name)
	}
	c := NewCollection(name)
	if err := s.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Delete removes a collection's file from disk.
func (s *Store) Delete(name string) error {
	err := os.Remove(s.path(name))
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("collection %q not found", name)
	}
	return err
}

// Rename changes a collection's display name and its backing file.
func (s *Store) Rename(oldName, newName string) error {
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("new name cannot be empty")
	}
	c, err := s.Load(oldName)
	if err != nil {
		return err
	}
	newPath := s.path(newName)
	if slugify(newName) != slugify(oldName) {
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("collection %q already exists", newName)
		}
	}
	oldPath := s.path(oldName)
	c.Info.Name = newName
	if err := s.Save(c); err != nil {
		return err
	}
	if oldPath != newPath {
		os.Remove(oldPath)
	}
	return nil
}
