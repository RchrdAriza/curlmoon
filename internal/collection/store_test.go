package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStore_CreateSaveLoad(t *testing.T) {
	s := NewStore(t.TempDir())

	c, err := s.Create("My Requests")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	c.AddItemAt(nil, NewRequestItem("Get todo", "GET", "https://example.com", nil, "", ""))
	if err := s.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := s.Load("My Requests")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Info.Name != "My Requests" {
		t.Errorf("expected name preserved, got %s", loaded.Info.Name)
	}
	if len(loaded.Item) != 1 || loaded.Item[0].Request.URL.Raw != "https://example.com" {
		t.Errorf("expected item to round-trip, got %v", loaded.Item)
	}
}

func TestStore_CreateDuplicateFails(t *testing.T) {
	s := NewStore(t.TempDir())
	if _, err := s.Create("Dup"); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := s.Create("Dup"); err == nil {
		t.Error("expected duplicate create to fail")
	}
}

func TestExampleCollections(t *testing.T) {
	cols := ExampleCollections()
	if len(cols) != 3 {
		t.Fatalf("expected 3 example collections, got %d", len(cols))
	}

	s := NewStore(t.TempDir())
	names, err := s.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected ExampleCollections to not touch the store, got %v", names)
	}
}

func TestStore_Import(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "imported.json")
	body := `{"info":{"name":"Imported API"},"item":[{"name":"Ping","request":{"method":"GET","url":{"raw":"https://example.com/ping"}}}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	s := NewStore(t.TempDir())
	c, err := s.Import(path)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if c.Info.Name != "Imported API" {
		t.Errorf("expected name Imported API, got %s", c.Info.Name)
	}

	loaded, err := s.Load("Imported API")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Item) != 1 || loaded.Item[0].Request.URL.Raw != "https://example.com/ping" {
		t.Errorf("expected imported item to round-trip, got %v", loaded.Item)
	}
}

func TestStore_ImportMissingNameFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noname.json")
	if err := os.WriteFile(path, []byte(`{"info":{},"item":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	s := NewStore(t.TempDir())
	if _, err := s.Import(path); err == nil {
		t.Error("expected import of collection without a name to fail")
	}
}

func TestStore_CreateEmptyNameFails(t *testing.T) {
	s := NewStore(t.TempDir())
	if _, err := s.Create("   "); err == nil {
		t.Error("expected empty name to fail")
	}
}

func TestStore_List(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("Beta")
	s.Create("Alpha")

	names, err := s.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Beta" {
		t.Errorf("expected sorted [Alpha Beta], got %v", names)
	}
}

func TestStore_LoadAll(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("One")
	s.Create("Two")

	cols, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(cols) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(cols))
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("Temp")

	if err := s.Delete("Temp"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := s.Load("Temp"); err == nil {
		t.Error("expected load after delete to fail")
	}
}

func TestStore_DeleteMissingFails(t *testing.T) {
	s := NewStore(t.TempDir())
	if err := s.Delete("Nope"); err == nil {
		t.Error("expected delete of missing collection to fail")
	}
}

func TestStore_Rename(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("Old Name")

	if err := s.Rename("Old Name", "New Name"); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	if _, err := s.Load("Old Name"); err == nil {
		t.Error("expected old name to be gone after rename")
	}
	loaded, err := s.Load("New Name")
	if err != nil {
		t.Fatalf("expected new name to load: %v", err)
	}
	if loaded.Info.Name != "New Name" {
		t.Errorf("expected Info.Name updated, got %s", loaded.Info.Name)
	}

	names, _ := s.List()
	if len(names) != 1 {
		t.Errorf("expected exactly one collection after rename, got %v", names)
	}
}

func TestStore_RenameToExistingFails(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("A")
	s.Create("B")

	if err := s.Rename("A", "B"); err == nil {
		t.Error("expected rename to an existing name to fail")
	}
}

func TestStore_RenameSameSlugSucceeds(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Create("Name")

	if err := s.Rename("Name", "  Name  "); err != nil {
		t.Fatalf("expected rename with same slug to succeed, got %v", err)
	}
}

func TestStore_Export(t *testing.T) {
	s := NewStore(t.TempDir())
	c, err := s.Create("Exportable")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	c.AddItemAt(nil, NewRequestItem("Ping", "GET", "https://example.com/ping", nil, "", ""))
	if err := s.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out.json")
	if err := s.Export("Exportable", dest); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var roundtrip Collection
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if roundtrip.Info.Name != "Exportable" {
		t.Errorf("expected name preserved, got %s", roundtrip.Info.Name)
	}
	if len(roundtrip.Item) != 1 || roundtrip.Item[0].Request.URL.Raw != "https://example.com/ping" {
		t.Errorf("expected item to round-trip, got %v", roundtrip.Item)
	}
}

func TestStore_ExportMissingCollectionFails(t *testing.T) {
	s := NewStore(t.TempDir())
	if err := s.Export("Nope", filepath.Join(t.TempDir(), "out.json")); err == nil {
		t.Error("expected export of missing collection to fail")
	}
}
