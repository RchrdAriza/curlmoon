package collection

import "testing"

func TestSession_SaveLoadRoundTrip(t *testing.T) {
	s := NewStore(t.TempDir())

	sess := &Session{
		Method:    "POST",
		URL:       "https://example.com",
		Headers:   []KeyVal{{Key: "X-Test", Value: "1"}},
		Params:    []KeyVal{{Key: "q", Value: "search"}},
		BodyType:  "JSON",
		Body:      `{"a":1}`,
		ActiveTab: 1,
	}
	if err := s.SaveSession(sess); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	loaded, err := s.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected session to load")
	}
	if loaded.Method != "POST" || loaded.URL != "https://example.com" {
		t.Errorf("expected method/url preserved, got %+v", loaded)
	}
	if len(loaded.Headers) != 1 || loaded.Headers[0].Key != "X-Test" {
		t.Errorf("expected headers preserved, got %v", loaded.Headers)
	}
	if loaded.Body != `{"a":1}` {
		t.Errorf("expected body preserved, got %s", loaded.Body)
	}
}

func TestSession_LoadMissingReturnsNil(t *testing.T) {
	s := NewStore(t.TempDir())

	sess, err := s.LoadSession()
	if err != nil {
		t.Fatalf("expected no error for missing session, got %v", err)
	}
	if sess != nil {
		t.Errorf("expected nil session when none saved, got %+v", sess)
	}
}
