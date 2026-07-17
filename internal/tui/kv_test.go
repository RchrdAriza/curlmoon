package tui

import "testing"

func TestParseKV_Basic(t *testing.T) {
	pairs := parseKV("Content-Type: application/json\nAuthorization: Bearer token")
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	if pairs[0].Key != "Content-Type" || pairs[0].Value != "application/json" {
		t.Errorf("unexpected first pair: %+v", pairs[0])
	}
	if pairs[1].Key != "Authorization" || pairs[1].Value != "Bearer token" {
		t.Errorf("unexpected second pair: %+v", pairs[1])
	}
}

func TestParseKV_BlankLinesIgnored(t *testing.T) {
	pairs := parseKV("\n\nX-Test: 1\n\n")
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
}

func TestParseKV_Empty(t *testing.T) {
	if pairs := parseKV(""); len(pairs) != 0 {
		t.Errorf("expected 0 pairs for empty text, got %d", len(pairs))
	}
}

func TestParseKV_NoColonIsKeyOnly(t *testing.T) {
	pairs := parseKV("justakey")
	if len(pairs) != 1 || pairs[0].Key != "justakey" || pairs[0].Value != "" {
		t.Errorf("unexpected pairs: %+v", pairs)
	}
}

func TestSerializeKV_RoundTrip(t *testing.T) {
	original := []KeyValuePair{{Key: "Accept", Value: "*/*"}, {Key: "X-Test", Value: "1"}}
	text := serializeKV(original)
	pairs := parseKV(text)
	if len(pairs) != len(original) {
		t.Fatalf("expected %d pairs after round-trip, got %d", len(original), len(pairs))
	}
	for i, p := range pairs {
		if p != original[i] {
			t.Errorf("pair %d mismatch: got %+v, want %+v", i, p, original[i])
		}
	}
}

func TestKVToMap(t *testing.T) {
	m := kvToMap([]KeyValuePair{{Key: "Accept", Value: "*/*"}})
	if m["Accept"] != "*/*" {
		t.Errorf("expected */*, got %s", m["Accept"])
	}
}

func TestKVToMap_EmptyKeySkipped(t *testing.T) {
	m := kvToMap([]KeyValuePair{{Key: "", Value: "x"}})
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}
