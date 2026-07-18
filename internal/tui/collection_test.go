package tui

import (
	"curlmoon/internal/collection"
	"testing"
)

func TestNewAppWithStore_NoAutomaticSeeding(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	a := NewAppWithStore(store)

	if len(a.collections) != 0 {
		t.Fatalf("expected no collections on an empty store, got %d", len(a.collections))
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected nothing persisted to disk, got %v", names)
	}
}

func TestNewAppWithStore_ExtraCollectionsAreSessionOnly(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	demo := collection.ExampleCollections()

	a := NewAppWithStore(store, demo...)
	if len(a.collections) != 3 {
		t.Fatalf("expected 3 in-memory collections, got %d", len(a.collections))
	}
	if len(a.sidebar) == 0 {
		t.Fatal("expected sidebar to be populated from collections")
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected demo collections to not be persisted, got %v", names)
	}
}

func TestNewAppWithStore_LoadsExistingCollections(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	c, _ := store.Create("My Stuff")
	c.AddItemAt(nil, collection.NewRequestItem("Ping", "GET", "https://example.com/ping", nil, "", ""))
	store.Save(c)

	a := NewAppWithStore(store)

	if len(a.collections) != 1 {
		t.Fatalf("expected the pre-existing collection to be loaded, got %d", len(a.collections))
	}

	var found bool
	for _, e := range a.sidebar {
		if e.name == "Ping" && e.url == "https://example.com/ping" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sidebar to contain the loaded request, got %+v", a.sidebar)
	}
}

func TestSidebar_CreateCollection(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	a := NewAppWithStore(store)
	before := len(a.collections)

	a.StartPrompt("newCollection", sidebarEntry{}, "")
	if a.promptMode != "newCollection" {
		t.Fatalf("expected newCollection prompt, got %q", a.promptMode)
	}

	a.promptText = "Team Alpha"
	a.ConfirmPrompt()

	if a.promptMode != "" {
		t.Errorf("expected prompt to close after confirm, got %q", a.promptMode)
	}
	if len(a.collections) != before+1 {
		t.Fatalf("expected a new collection to be added, got %d", len(a.collections))
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
	store.Create("My Stuff")
	a := NewAppWithStore(store)
	a.urlValue = "https://example.com/save-me"
	a.sidebarSel = 1 // 0 is the "Collections" header, 1 is the first collection root

	a.StartPrompt("newRequest", sidebarEntry{collIdx: a.sidebar[a.sidebarSel].collIdx}, "")
	if a.promptMode != "newRequest" {
		t.Fatalf("expected newRequest prompt, got %q", a.promptMode)
	}

	a.promptText = "Saved Req"
	a.ConfirmPrompt()

	var found bool
	for _, e := range a.sidebar {
		if e.name == "Saved Req" && e.url == "https://example.com/save-me" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected saved request to appear in sidebar, got %+v", a.sidebar)
	}
}

func TestSidebar_RenameCollection(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	store.Create("Old Name")
	a := NewAppWithStore(store)
	a.sidebarSel = 1 // 0 is the "Collections" header

	sel := a.sidebar[a.sidebarSel]
	a.StartPrompt("rename", sel, sel.name)
	if a.promptMode != "rename" {
		t.Fatalf("expected rename prompt, got %q", a.promptMode)
	}

	a.promptText = "New Name"
	a.ConfirmPrompt()

	if a.collections[0].Info.Name != "New Name" {
		t.Errorf("expected renamed collection, got %q", a.collections[0].Info.Name)
	}
	if _, err := store.Load("Old Name"); err == nil {
		t.Error("expected old collection file to be gone")
	}
}

func TestSidebar_DeleteCollectionWithConfirm(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	store.Create("Doomed")
	a := NewAppWithStore(store)
	a.sidebarSel = 1 // 0 is the "Collections" header

	sel := a.sidebar[a.sidebarSel]
	a.StartPrompt("confirmDelete", sel, "")
	if a.promptMode != "confirmDelete" {
		t.Fatalf("expected confirmDelete prompt, got %q", a.promptMode)
	}

	a.ConfirmPrompt()

	if len(a.collections) != 0 {
		t.Errorf("expected collection removed from model, got %d", len(a.collections))
	}
	if _, err := store.Load("Doomed"); err == nil {
		t.Error("expected collection file deleted from disk")
	}
}

func TestSidebar_DeleteCollectionCancelled(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	store.Create("Safe")
	a := NewAppWithStore(store)
	a.sidebarSel = 1 // 0 is the "Collections" header

	sel := a.sidebar[a.sidebarSel]
	a.StartPrompt("confirmDelete", sel, "")
	a.CancelPrompt()

	if len(a.collections) != 1 {
		t.Errorf("expected collection preserved after cancel, got %d", len(a.collections))
	}
	if _, err := store.Load("Safe"); err != nil {
		t.Errorf("expected collection file preserved, got %v", err)
	}
}

func TestSessionRoundTrip_ThroughApp(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	a := NewAppWithStore(store)
	a.urlValue = "https://example.com/session"
	a.methodIndex = 1 // POST
	a.bodyType = 1    // JSON
	a.bodyText = `{"x":1}`
	a.headersText = "X-Test: abc"

	a.saveSession()

	a2 := NewAppWithStore(store)
	if a2.urlValue != "https://example.com/session" {
		t.Errorf("expected URL restored, got %s", a2.urlValue)
	}
	if a2.methodIndex != 1 {
		t.Errorf("expected method restored, got %d", a2.methodIndex)
	}
	if a2.bodyType != 1 {
		t.Errorf("expected bodyType restored, got %d", a2.bodyType)
	}
	if a2.bodyText != `{"x":1}` {
		t.Errorf("expected body restored, got %s", a2.bodyText)
	}
	pairs := parseKV(a2.headersText)
	if len(pairs) != 1 || pairs[0].Key != "X-Test" || pairs[0].Value != "abc" {
		t.Errorf("expected header restored, got %+v", pairs)
	}
}
