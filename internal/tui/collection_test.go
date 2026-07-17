package tui

import (
	"curlmoon/internal/collection"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelWithStore_SeedsDefaultCollections(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	m := NewModelWithStore(store)

	if len(m.collections) != 3 {
		t.Fatalf("expected 3 seeded collections, got %d", len(m.collections))
	}
	if len(m.sidebar) == 0 {
		t.Fatal("expected sidebar to be populated from collections")
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("expected seeded collections persisted to disk, got %v", names)
	}
}

func TestNewModelWithStore_LoadsExistingCollections(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	c, _ := store.Create("My Stuff")
	c.AddItemAt(nil, collection.NewRequestItem("Ping", "GET", "https://example.com/ping", nil, "", ""))
	store.Save(c)

	m := NewModelWithStore(store)

	if len(m.collections) != 1 {
		t.Fatalf("expected the pre-existing collection to be loaded, got %d", len(m.collections))
	}

	var found bool
	for _, e := range m.sidebar {
		if e.name == "Ping" && e.url == "https://example.com/ping" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sidebar to contain the loaded request, got %+v", m.sidebar)
	}
}

func TestSidebar_CreateCollection(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	m := NewModelWithStore(store)
	m.activePanel = panelSidebar
	before := len(m.collections)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m2 := result.(Model)
	if m2.promptMode != "newCollection" {
		t.Fatalf("expected newCollection prompt, got %q", m2.promptMode)
	}

	for _, r := range "Team Alpha" {
		result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m2 = result.(Model)
	}
	result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := result.(Model)

	if m3.promptMode != "" {
		t.Errorf("expected prompt to close after Enter, got %q", m3.promptMode)
	}
	if len(m3.collections) != before+1 {
		t.Fatalf("expected a new collection to be added, got %d", len(m3.collections))
	}

	names, _ := store.List()
	var has bool
	for _, n := range names {
		if n == "Team Alpha" {
			has = true
		}
	}
	if !has {
		t.Errorf("expected new collection persisted to disk, got %v", names)
	}
}

func TestSidebar_SaveRequestIntoCollection(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	m := NewModelWithStore(store)
	m = m.initLayout(100, 30)
	m.urlInput.SetValue("https://example.com/save-me")
	m.activePanel = panelSidebar
	m.sidebarSel = 0 // first entry is a collection root

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m2 := result.(Model)
	if m2.promptMode != "newRequest" {
		t.Fatalf("expected newRequest prompt, got %q", m2.promptMode)
	}

	for _, r := range "Saved Req" {
		result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m2 = result.(Model)
	}
	result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := result.(Model)

	var found bool
	for _, e := range m3.sidebar {
		if e.name == "Saved Req" && e.url == "https://example.com/save-me" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected saved request to appear in sidebar, got %+v", m3.sidebar)
	}
}

func TestSidebar_RenameCollection(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	store.Create("Old Name")
	m := NewModelWithStore(store)
	m.activePanel = panelSidebar
	m.sidebarSel = 0

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m2 := result.(Model)
	if m2.promptMode != "rename" {
		t.Fatalf("expected rename prompt, got %q", m2.promptMode)
	}

	// clear the prefilled name, then type a new one
	m2.promptInput.SetValue("")
	for _, r := range "New Name" {
		result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m2 = result.(Model)
	}
	result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := result.(Model)

	if m3.collections[0].Info.Name != "New Name" {
		t.Errorf("expected renamed collection, got %q", m3.collections[0].Info.Name)
	}
	if _, err := store.Load("Old Name"); err == nil {
		t.Error("expected old collection file to be gone")
	}
}

func TestSidebar_DeleteCollectionWithConfirm(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	store.Create("Doomed")
	m := NewModelWithStore(store)
	m.activePanel = panelSidebar
	m.sidebarSel = 0

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m2 := result.(Model)
	if m2.promptMode != "confirmDelete" {
		t.Fatalf("expected confirmDelete prompt, got %q", m2.promptMode)
	}

	result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m3 := result.(Model)

	if len(m3.collections) != 0 {
		t.Errorf("expected collection removed from model, got %d", len(m3.collections))
	}
	if _, err := store.Load("Doomed"); err == nil {
		t.Error("expected collection file deleted from disk")
	}
}

func TestSidebar_DeleteCollectionCancelled(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	store.Create("Safe")
	m := NewModelWithStore(store)
	m.activePanel = panelSidebar
	m.sidebarSel = 0

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m2 := result.(Model)

	result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m3 := result.(Model)

	if len(m3.collections) != 1 {
		t.Errorf("expected collection preserved after cancel, got %d", len(m3.collections))
	}
	if _, err := store.Load("Safe"); err != nil {
		t.Errorf("expected collection file preserved, got %v", err)
	}
}

func TestSessionRoundTrip_ThroughModel(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	m := NewModelWithStore(store)
	m.urlInput.SetValue("https://example.com/session")
	m.methodIndex = 1 // POST
	m.bodyType = 1     // JSON
	m.bodyEditor.SetValue(`{"x":1}`)
	m.headers.Rows[0].key.SetValue("X-Test")
	m.headers.Rows[0].value.SetValue("abc")

	m.saveSession()

	m2 := NewModelWithStore(store)
	if m2.urlInput.Value() != "https://example.com/session" {
		t.Errorf("expected URL restored, got %s", m2.urlInput.Value())
	}
	if m2.methodIndex != 1 {
		t.Errorf("expected method restored, got %d", m2.methodIndex)
	}
	if m2.bodyType != 1 {
		t.Errorf("expected bodyType restored, got %d", m2.bodyType)
	}
	if m2.bodyEditor.Value() != `{"x":1}` {
		t.Errorf("expected body restored, got %s", m2.bodyEditor.Value())
	}
	if m2.headers.Rows[0].key.Value() != "X-Test" {
		t.Errorf("expected header restored, got %s", m2.headers.Rows[0].key.Value())
	}
}
