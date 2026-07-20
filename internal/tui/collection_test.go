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

func TestSelectSidebarEntry_LoadsOwnTabsAndDoesNotInherit(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	c, _ := store.Create("Stuff")
	full := collection.NewRequestItem("Full", "POST", "https://example.com/a",
		map[string]string{"X-Token": "abc"}, `{"hi":true}`, "JSON")
	full.Event = append(full.Event, collection.Event{
		Listen: "prerequest", Script: collection.NewScript("pm.environment.set('x', 1)"),
	})
	c.AddItemAt(nil, full)
	c.AddItemAt(nil, collection.NewRequestItem("Bare", "GET", "https://example.com/b", nil, "", ""))
	store.Save(c)

	a := NewAppWithStore(store)

	selectByName := func(name string) {
		for i, e := range a.sidebar {
			if e.name == name && !e.isFolder {
				a.sidebarSel = i
				if !a.SelectSidebarEntry() {
					t.Fatalf("expected %q to load as a request", name)
				}
				return
			}
		}
		t.Fatalf("sidebar entry %q not found in %+v", name, a.sidebar)
	}

	selectByName("Full")
	if a.headersText != "X-Token: abc" {
		t.Errorf("expected Full's headers loaded, got %q", a.headersText)
	}
	if a.bodyText != `{"hi":true}` || a.bodyType != 1 {
		t.Errorf("expected Full's JSON body loaded, got type=%d body=%q", a.bodyType, a.bodyText)
	}
	pre, _ := parseScripts(a.scriptsText)
	if pre != "pm.environment.set('x', 1)" {
		t.Errorf("expected Full's pre-request script loaded, got %q", pre)
	}

	// Switching to a request with no headers/body/scripts must not inherit
	// Full's tab content.
	selectByName("Bare")
	if a.headersText != "" {
		t.Errorf("expected headers reset for Bare, got %q", a.headersText)
	}
	if a.bodyText != "" || a.bodyType != 0 {
		t.Errorf("expected body reset for Bare, got type=%d body=%q", a.bodyType, a.bodyText)
	}
	if a.scriptsText != defaultScriptsText {
		t.Errorf("expected scripts reset to default for Bare, got %q", a.scriptsText)
	}
}

// selectRequestByName finds a non-folder sidebar entry and loads it.
func selectRequestByName(t *testing.T, a *App, name string) {
	t.Helper()
	for i, e := range a.sidebar {
		if e.name == name && !e.isFolder {
			a.sidebarSel = i
			if !a.SelectSidebarEntry() {
				t.Fatalf("expected %q to load", name)
			}
			return
		}
	}
	t.Fatalf("sidebar entry %q not found", name)
}

func TestSaveActiveRequest_PersistsEditsToDisk(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	c, _ := store.Create("C")
	c.AddItemAt(nil, collection.NewRequestItem("R", "GET", "https://ex.com/r", nil, "", ""))
	store.Save(c)

	a := NewAppWithStore(store)
	selectRequestByName(t, a, "R")

	// Edit several tabs, as the editor would.
	for i, m := range methods {
		if m == "POST" {
			a.methodIndex = i
		}
	}
	a.bodyType = 1 // JSON
	a.bodyText = `{"a":1}`
	a.headersText = "X-Test: yes"
	a.markDirty()

	if !a.dirty {
		t.Fatal("expected request to be dirty after edits")
	}
	if !a.SaveActiveRequest() {
		t.Fatal("expected SaveActiveRequest to succeed")
	}
	if a.dirty {
		t.Error("expected dirty cleared after save")
	}

	// A fresh app loading the same store must see the persisted edits.
	b := NewAppWithStore(store)
	selectRequestByName(t, b, "R")
	if methods[b.methodIndex] != "POST" {
		t.Errorf("method not persisted, got %s", methods[b.methodIndex])
	}
	if b.bodyText != `{"a":1}` || b.bodyType != 1 {
		t.Errorf("body not persisted, got type=%d body=%q", b.bodyType, b.bodyText)
	}
	if got := kvToMap(parseKV(b.headersText)); got["X-Test"] != "yes" {
		t.Errorf("header not persisted, got %q", b.headersText)
	}
}

func TestSaveActiveRequest_NoOpWithoutActiveItem(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	a := NewAppWithStore(store)
	a.activeCollIdx = -1
	if a.SaveActiveRequest() {
		t.Error("expected SaveActiveRequest to be a no-op with no active request")
	}
}

func TestCancelEdit_RevertsSnapshot(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://original"
	a.bodyText = "orig-body"

	if !a.EnterEditURL() {
		t.Fatal("expected to enter URL edit mode")
	}
	if a.EnterEditURL() {
		t.Error("expected EnterEditURL to no-op while already editing")
	}

	a.urlValue = "https://changed"
	a.bodyText = "changed-body"
	a.markDirty()

	a.CancelEdit()
	if a.urlEditing {
		t.Error("expected edit mode to end after cancel")
	}
	if a.dirty {
		t.Error("expected not dirty after cancel")
	}
	if a.urlValue != "https://original" || a.bodyText != "orig-body" {
		t.Errorf("cancel did not revert, got url=%q body=%q", a.urlValue, a.bodyText)
	}
}

func TestLoadRequestClearsDirtyAndEditMode(t *testing.T) {
	store := collection.NewStore(t.TempDir())
	c, _ := store.Create("C")
	c.AddItemAt(nil, collection.NewRequestItem("R", "GET", "https://ex.com/r", nil, "", ""))
	store.Save(c)
	a := NewAppWithStore(store)

	a.dirty = true
	a.urlEditing = true
	selectRequestByName(t, a, "R")
	if a.dirty || a.urlEditing {
		t.Errorf("loading a request should reset dirty/urlEditing, got dirty=%v editing=%v", a.dirty, a.urlEditing)
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
