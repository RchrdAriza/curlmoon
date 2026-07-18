// Package config loads curlmoon's user-editable settings: keybindings and
// the active color theme. Both live as small JSON files under the app's
// base directory (~/.curlmoon by default) and fall back to sane defaults
// when missing or invalid, so a fresh install needs no configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/gocui"
)

// KeyBinding is the JSON-serializable form of a gocui key + modifier, e.g.
// {"key": "ctrl+r"} or {"key": "q"}.
type KeyBinding struct {
	Key string `json:"key"`
}

// Keymap maps an action name (e.g. "sendRequest") to the key that triggers
// it. Action names are the source of truth defined by DefaultKeymap.
type Keymap map[string]KeyBinding

// DefaultKeymap returns curlmoon's built-in keybindings, matching the
// hardcoded values curlmoon shipped with before keybindings became
// configurable. This is also what LoadKeymap merges a user's file on top of.
func DefaultKeymap() Keymap {
	return Keymap{
		"quit":       {"q"},
		"cycleFocus": {"tab"},

		"jumpSidebar":  {"ctrl+s"},
		"jumpURL":      {"ctrl+u"},
		"jumpResponse": {"ctrl+e"},
		"jumpContent":  {"ctrl+b"},
		"sendRequest":  {"ctrl+r"},

		"sidebarUp":            {"up"},
		"sidebarDown":          {"down"},
		"sidebarEnter":         {"enter"},
		"sidebarNewCollection": {"n"},
		"sidebarNewRequest":    {"a"},
		"sidebarRename":        {"r"},
		"sidebarDelete":        {"d"},
		"sidebarEditVars":      {"v"},
		"sidebarExport":        {"x"},
		"sidebarImport":        {"i"},

		"urlMethodUp":      {"up"},
		"urlMethodDown":    {"down"},
		"urlSwitchTabPrev": {"ctrl+p"},
		"urlSwitchTabNext": {"ctrl+n"},
		"urlHome":          {"home"},
		"urlEnd":           {"end"},
		"urlEnterContent":  {"enter"},
		"cycleBodyType":    {"ctrl+y"},

		"contentEsc": {"esc"},

		"responseScrollUp":   {"up"},
		"responseScrollDown": {"down"},
		"responsePageUp":     {"pgup"},
		"responsePageDown":   {"pgdn"},

		"toggleHelp":    {"ctrl+/"},
		"toggleCodegen": {"ctrl+g"},
		"toggleTheme":   {"ctrl+t"},
	}
}

// Display renders a KeyBinding's key string in a human-friendly form for
// help text, e.g. "ctrl+r" -> "Ctrl+R", "up" -> "↑".
func (kb KeyBinding) Display() string {
	parts := strings.Split(kb.Key, "+")
	for i, p := range parts {
		switch p {
		case "up":
			parts[i] = "↑"
		case "down":
			parts[i] = "↓"
		case "left":
			parts[i] = "←"
		case "right":
			parts[i] = "→"
		case "enter", "return":
			parts[i] = "Enter"
		case "esc", "escape":
			parts[i] = "Esc"
		case "tab":
			parts[i] = "Tab"
		case "home":
			parts[i] = "Home"
		case "end":
			parts[i] = "End"
		case "pgup":
			parts[i] = "PgUp"
		case "pgdn":
			parts[i] = "PgDn"
		case "ctrl":
			parts[i] = "Ctrl"
		case "alt":
			parts[i] = "Alt"
		default:
			if len(p) == 1 {
				parts[i] = strings.ToUpper(p)
			} else if p != "" {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
	}
	return strings.Join(parts, "+")
}

// keymapPath returns the path to the keybindings file under baseDir.
func keymapPath(baseDir string) string {
	return filepath.Join(baseDir, "keybindings.json")
}

// LoadKeymap reads baseDir/keybindings.json and merges it over
// DefaultKeymap: a missing file, an unparsable file, or an action absent
// from the file all fall back to the default for that action.
func LoadKeymap(baseDir string) Keymap {
	km := DefaultKeymap()
	data, err := os.ReadFile(keymapPath(baseDir))
	if err != nil {
		return km
	}
	var overrides Keymap
	if err := json.Unmarshal(data, &overrides); err != nil {
		return km
	}
	for action, kb := range overrides {
		if strings.TrimSpace(kb.Key) == "" {
			continue
		}
		km[action] = kb
	}
	return km
}

// Key resolves the key bound to action, falling back to def (typically the
// default binding) if the action is missing or its key string doesn't parse.
func (km Keymap) Key(action string) (interface{}, gocui.Modifier) {
	kb, ok := km[action]
	if !ok {
		kb = DefaultKeymap()[action]
	}
	key, mod, err := ParseKey(kb.Key)
	if err != nil {
		key, mod, _ = ParseKey(DefaultKeymap()[action].Key)
	}
	return key, mod
}

// DisplayKey returns the human-friendly rendering of the key bound to
// action, falling back to the default binding's display if action is unset.
func (km Keymap) DisplayKey(action string) string {
	kb, ok := km[action]
	if !ok {
		kb = DefaultKeymap()[action]
	}
	return kb.Display()
}

// ParseKey translates a human-readable key string ("q", "ctrl+r", "up", ...)
// into the (key, modifier) pair gocui.SetKeybinding expects. key is either a
// gocui.Key or a rune, matching gocui's own interface{} key parameter.
func ParseKey(s string) (interface{}, gocui.Modifier, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, gocui.ModNone, fmt.Errorf("config: empty key")
	}
	lower := strings.ToLower(s)

	if rest, ok := strings.CutPrefix(lower, "ctrl+"); ok {
		return parseCtrlKey(rest)
	}
	if rest, ok := strings.CutPrefix(lower, "alt+"); ok {
		key, _, err := ParseKey(rest)
		if err != nil {
			return nil, gocui.ModNone, err
		}
		return key, gocui.ModAlt, nil
	}

	switch lower {
	case "up":
		return gocui.KeyArrowUp, gocui.ModNone, nil
	case "down":
		return gocui.KeyArrowDown, gocui.ModNone, nil
	case "left":
		return gocui.KeyArrowLeft, gocui.ModNone, nil
	case "right":
		return gocui.KeyArrowRight, gocui.ModNone, nil
	case "enter", "return":
		return gocui.KeyEnter, gocui.ModNone, nil
	case "esc", "escape":
		return gocui.KeyEsc, gocui.ModNone, nil
	case "tab":
		return gocui.KeyTab, gocui.ModNone, nil
	case "space":
		return gocui.KeySpace, gocui.ModNone, nil
	case "home":
		return gocui.KeyHome, gocui.ModNone, nil
	case "end":
		return gocui.KeyEnd, gocui.ModNone, nil
	case "pgup":
		return gocui.KeyPgup, gocui.ModNone, nil
	case "pgdn":
		return gocui.KeyPgdn, gocui.ModNone, nil
	case "backspace":
		return gocui.KeyBackspace2, gocui.ModNone, nil
	case "delete":
		return gocui.KeyDelete, gocui.ModNone, nil
	}

	r := []rune(s)
	if len(r) == 1 {
		return r[0], gocui.ModNone, nil
	}
	return nil, gocui.ModNone, fmt.Errorf("config: unrecognized key %q", s)
}

func parseCtrlKey(name string) (interface{}, gocui.Modifier, error) {
	switch name {
	case "a":
		return gocui.KeyCtrlA, gocui.ModNone, nil
	case "b":
		return gocui.KeyCtrlB, gocui.ModNone, nil
	case "c":
		return gocui.KeyCtrlC, gocui.ModNone, nil
	case "d":
		return gocui.KeyCtrlD, gocui.ModNone, nil
	case "e":
		return gocui.KeyCtrlE, gocui.ModNone, nil
	case "f":
		return gocui.KeyCtrlF, gocui.ModNone, nil
	case "g":
		return gocui.KeyCtrlG, gocui.ModNone, nil
	case "h":
		return gocui.KeyCtrlH, gocui.ModNone, nil
	case "i":
		return gocui.KeyCtrlI, gocui.ModNone, nil
	case "j":
		return gocui.KeyCtrlJ, gocui.ModNone, nil
	case "k":
		return gocui.KeyCtrlK, gocui.ModNone, nil
	case "l":
		return gocui.KeyCtrlL, gocui.ModNone, nil
	case "m":
		return gocui.KeyCtrlM, gocui.ModNone, nil
	case "n":
		return gocui.KeyCtrlN, gocui.ModNone, nil
	case "o":
		return gocui.KeyCtrlO, gocui.ModNone, nil
	case "p":
		return gocui.KeyCtrlP, gocui.ModNone, nil
	case "q":
		return gocui.KeyCtrlQ, gocui.ModNone, nil
	case "r":
		return gocui.KeyCtrlR, gocui.ModNone, nil
	case "s":
		return gocui.KeyCtrlS, gocui.ModNone, nil
	case "t":
		return gocui.KeyCtrlT, gocui.ModNone, nil
	case "u":
		return gocui.KeyCtrlU, gocui.ModNone, nil
	case "v":
		return gocui.KeyCtrlV, gocui.ModNone, nil
	case "w":
		return gocui.KeyCtrlW, gocui.ModNone, nil
	case "x":
		return gocui.KeyCtrlX, gocui.ModNone, nil
	case "y":
		return gocui.KeyCtrlY, gocui.ModNone, nil
	case "z":
		return gocui.KeyCtrlZ, gocui.ModNone, nil
	case "/":
		return gocui.KeyCtrlSlash, gocui.ModNone, nil
	case "\\":
		return gocui.KeyCtrlBackslash, gocui.ModNone, nil
	case "space":
		return gocui.KeyCtrlSpace, gocui.ModNone, nil
	}
	return nil, gocui.ModNone, fmt.Errorf("config: unrecognized ctrl key %q", name)
}

// AppConfig holds settings other than keybindings — currently just the
// active theme name ("dark" or "light").
type AppConfig struct {
	Theme string `json:"theme,omitempty"`
}

func configPath(baseDir string) string {
	return filepath.Join(baseDir, "config.json")
}

// LoadTheme reads baseDir/config.json and returns the saved theme name, or
// "" if none is set or the file doesn't exist.
func LoadTheme(baseDir string) string {
	data, err := os.ReadFile(configPath(baseDir))
	if err != nil {
		return ""
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.Theme
}

// SaveTheme persists the active theme name to baseDir/config.json.
func SaveTheme(baseDir, theme string) error {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(AppConfig{Theme: theme}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(baseDir), data, 0o644)
}
