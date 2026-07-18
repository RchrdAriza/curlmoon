package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/gocui"
)

func TestParseKey_Letters(t *testing.T) {
	key, mod, err := ParseKey("q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != rune('q') || mod != gocui.ModNone {
		t.Errorf("got key=%v mod=%v", key, mod)
	}
}

func TestParseKey_Ctrl(t *testing.T) {
	key, mod, err := ParseKey("ctrl+r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != gocui.KeyCtrlR || mod != gocui.ModNone {
		t.Errorf("got key=%v mod=%v", key, mod)
	}
}

func TestParseKey_Named(t *testing.T) {
	cases := map[string]interface{}{
		"up":    gocui.KeyArrowUp,
		"esc":   gocui.KeyEsc,
		"enter": gocui.KeyEnter,
		"tab":   gocui.KeyTab,
	}
	for name, want := range cases {
		key, _, err := ParseKey(name)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if key != want {
			t.Errorf("%s: got %v, want %v", name, key, want)
		}
	}
}

func TestParseKey_Invalid(t *testing.T) {
	if _, _, err := ParseKey(""); err == nil {
		t.Error("expected error for empty key")
	}
	if _, _, err := ParseKey("ctrl+notakey"); err == nil {
		t.Error("expected error for unrecognized ctrl key")
	}
}

func TestDefaultKeymap_AllKeysParse(t *testing.T) {
	for action, kb := range DefaultKeymap() {
		if _, _, err := ParseKey(kb.Key); err != nil {
			t.Errorf("action %q has unparsable default key %q: %v", action, kb.Key, err)
		}
	}
}

func TestLoadKeymap_MissingFileFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	km := LoadKeymap(dir)
	if km["sendRequest"].Key != "ctrl+r" {
		t.Errorf("expected default sendRequest binding, got %+v", km["sendRequest"])
	}
}

func TestLoadKeymap_OverridesMergeOverDefaults(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keybindings.json"), []byte(`{"sendRequest": {"key": "ctrl+x"}}`), 0o644)
	km := LoadKeymap(dir)
	if km["sendRequest"].Key != "ctrl+x" {
		t.Errorf("expected overridden sendRequest binding, got %+v", km["sendRequest"])
	}
	if km["quit"].Key != "q" {
		t.Errorf("expected default quit binding to survive merge, got %+v", km["quit"])
	}
}

func TestLoadKeymap_InvalidJSONFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keybindings.json"), []byte(`not json`), 0o644)
	km := LoadKeymap(dir)
	if km["quit"].Key != "q" {
		t.Errorf("expected default quit binding, got %+v", km["quit"])
	}
}

func TestKeymap_Key_FallsBackOnBadOverride(t *testing.T) {
	km := DefaultKeymap()
	km["quit"] = KeyBinding{Key: "not-a-key!!"}
	key, _ := km.Key("quit")
	if key != rune('q') {
		t.Errorf("expected fallback to default quit key, got %v", key)
	}
}

func TestKeyBinding_Display(t *testing.T) {
	cases := map[string]string{
		"q":      "Q",
		"ctrl+r": "Ctrl+R",
		"up":     "↑",
		"esc":    "Esc",
		"ctrl+/": "Ctrl+/",
		"pgdn":   "PgDn",
	}
	for key, want := range cases {
		got := KeyBinding{Key: key}.Display()
		if got != want {
			t.Errorf("Display(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestKeymap_DisplayKey_FallsBackToDefault(t *testing.T) {
	km := Keymap{}
	if got := km.DisplayKey("quit"); got != "Q" {
		t.Errorf("expected default quit display Q, got %q", got)
	}
}

func TestSaveAndLoadTheme(t *testing.T) {
	dir := t.TempDir()
	if theme := LoadTheme(dir); theme != "" {
		t.Errorf("expected empty theme before saving, got %q", theme)
	}
	if err := SaveTheme(dir, "light"); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}
	if theme := LoadTheme(dir); theme != "light" {
		t.Errorf("expected light theme, got %q", theme)
	}
}
