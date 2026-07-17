package tui

import "strings"

// KeyValuePair is a single header or query param entry.
type KeyValuePair struct {
	Key   string
	Value string
}

// parseKV parses text where each non-blank line is "Key: Value" into pairs.
// Lines without a colon are treated as a key with an empty value.
func parseKV(text string) []KeyValuePair {
	var pairs []KeyValuePair
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, _ := strings.Cut(line, ":")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" && v == "" {
			continue
		}
		pairs = append(pairs, KeyValuePair{Key: k, Value: v})
	}
	return pairs
}

// serializeKV renders pairs back into "Key: Value" lines, one per row.
func serializeKV(pairs []KeyValuePair) string {
	lines := make([]string, len(pairs))
	for i, p := range pairs {
		lines[i] = p.Key + ": " + p.Value
	}
	return strings.Join(lines, "\n")
}

// kvToMap collapses pairs into a map, keeping the last value for duplicate keys.
func kvToMap(pairs []KeyValuePair) map[string]string {
	m := make(map[string]string)
	for _, p := range pairs {
		if p.Key != "" {
			m[p.Key] = p.Value
		}
	}
	return m
}
