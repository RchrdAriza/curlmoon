// Package environment implements Postman-style environments: named sets of
// {{variable}} substitutions that get resolved into requests at send time.
package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// KeyVal is a single environment variable. Disabled entries are kept (so the
// editor can round-trip them) but excluded from resolution.
type KeyVal struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// Environment is a named collection of variables, e.g. "local" or "prod".
type Environment struct {
	Name   string   `json:"name"`
	Values []KeyVal `json:"values,omitempty"`
}

// Vars collapses the enabled key/value pairs into a lookup map.
func (e *Environment) Vars() map[string]string {
	m := make(map[string]string)
	for _, kv := range e.Values {
		if kv.Enabled && kv.Key != "" {
			m[kv.Key] = kv.Value
		}
	}
	return m
}

var varRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// Resolve replaces every {{key}} occurrence in text with vars[key]. Tokens
// with no matching variable are left untouched.
func Resolve(text string, vars map[string]string) string {
	if len(vars) == 0 || !strings.Contains(text, "{{") {
		return text
	}
	return varRe.ReplaceAllStringFunc(text, func(tok string) string {
		key := varRe.FindStringSubmatch(tok)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		return tok
	})
}

// Store persists environments as JSON files under BaseDir/environments, plus
// a single active_environment file naming the currently active one.
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

func (s *Store) dir() string {
	return filepath.Join(s.BaseDir, "environments")
}

func (s *Store) ensureDir() error {
	return os.MkdirAll(s.dir(), 0o755)
}

var slugRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "environment"
	}
	return s
}

func (s *Store) path(name string) string {
	return filepath.Join(s.dir(), slugify(name)+".json")
}

// LoadAll reads every environment file, sorted alphabetically by name.
func (s *Store) LoadAll() ([]*Environment, error) {
	if err := s.ensureDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		return nil, err
	}
	var envs []*Environment
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir(), e.Name()))
		if err != nil {
			continue
		}
		var env Environment
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		envs = append(envs, &env)
	}
	sort.Slice(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })
	return envs, nil
}

// Save writes env to disk, keyed by its current Name.
func (s *Store) Save(env *Environment) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(env.Name), data, 0o644)
}

// Create makes a new empty environment and persists it.
func (s *Store) Create(name string) (*Environment, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("environment name cannot be empty")
	}
	if _, err := os.Stat(s.path(name)); err == nil {
		return nil, fmt.Errorf("environment %q already exists", name)
	}
	env := &Environment{Name: name}
	if err := s.Save(env); err != nil {
		return nil, err
	}
	return env, nil
}

// Delete removes an environment's file from disk.
func (s *Store) Delete(name string) error {
	err := os.Remove(s.path(name))
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("environment %q not found", name)
	}
	return err
}

// Rename changes an environment's display name and its backing file.
func (s *Store) Rename(oldName, newName string) error {
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("new name cannot be empty")
	}
	oldPath := s.path(oldName)
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return err
	}
	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}
	newPath := s.path(newName)
	if slugify(newName) != slugify(oldName) {
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("environment %q already exists", newName)
		}
	}
	env.Name = newName
	if err := s.Save(&env); err != nil {
		return err
	}
	if oldPath != newPath {
		os.Remove(oldPath)
	}
	return nil
}

// ParseDotenv parses KEY=VALUE lines in the style of a .env file: blank
// lines and lines starting with '#' are skipped, an optional leading
// "export " is stripped, and values may be wrapped in matching single or
// double quotes.
func ParseDotenv(data []byte) ([]KeyVal, error) {
	var values []KeyVal
	for i, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE, got %q", i+1, rawLine)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", i+1)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		values = append(values, KeyVal{Key: key, Value: value, Enabled: true})
	}
	return values, nil
}

// ImportDotenv reads a .env file and persists it as an environment named
// after the file (its base name without extension), overwriting any
// existing environment with that name.
func (s *Store) ImportDotenv(path string) (*Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values, err := ParseDotenv(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	env := &Environment{Name: name, Values: values}
	if err := s.Save(env); err != nil {
		return nil, err
	}
	return env, nil
}

func (s *Store) activePath() string {
	return filepath.Join(s.BaseDir, "active_environment.json")
}

// SetActive persists which environment name is currently active. An empty
// name clears the active environment.
func (s *Store) SetActive(name string) error {
	if err := os.MkdirAll(s.BaseDir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Name string `json:"name"`
	}{Name: name})
	if err != nil {
		return err
	}
	return os.WriteFile(s.activePath(), data, 0o644)
}

// LoadActive returns the persisted active environment name, or "" if none.
func (s *Store) LoadActive() (string, error) {
	data, err := os.ReadFile(s.activePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var v struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}
	return v.Name, nil
}
