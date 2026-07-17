package history

import "testing"

func TestStore_AddAndLoad(t *testing.T) {
	s := NewStore(t.TempDir())

	entries, err := s.Add(Entry{Method: "GET", URL: "https://example.com", StatusCode: 200})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].URL != "https://example.com" {
		t.Fatalf("expected persisted entry, got %+v", loaded)
	}
}

func TestStore_NewestFirst(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Add(Entry{Method: "GET", URL: "https://first.com"})
	entries, _ := s.Add(Entry{Method: "GET", URL: "https://second.com"})

	if entries[0].URL != "https://second.com" {
		t.Errorf("expected newest entry first, got %+v", entries)
	}
}

func TestStore_CapsAtMaxEntries(t *testing.T) {
	s := NewStore(t.TempDir())
	var entries []Entry
	for i := 0; i < MaxEntries+10; i++ {
		var err error
		entries, err = s.Add(Entry{Method: "GET", URL: "https://example.com"})
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	if len(entries) != MaxEntries {
		t.Errorf("expected history capped at %d entries, got %d", MaxEntries, len(entries))
	}
}

func TestStore_LoadEmpty(t *testing.T) {
	s := NewStore(t.TempDir())
	entries, err := s.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for fresh store, got %+v", entries)
	}
}
