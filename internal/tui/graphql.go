package tui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// graphqlVariablesDelim marks where a GraphQL query ends and its JSON
// variables begin inside the Body tab's text, e.g.:
//
//	query { me { id } }
//
//	### variables
//	{ "id": 1 }
const graphqlVariablesDelim = "### variables"

// parseGraphQLBody splits the Body tab's raw text into a query and its raw
// (still-JSON-text) variables. A missing delimiter means the whole text is
// the query with no variables.
func parseGraphQLBody(text string) (query, variablesJSON string) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), graphqlVariablesDelim) {
			query = strings.TrimSpace(strings.Join(lines[:i], "\n"))
			variablesJSON = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
			return
		}
	}
	return strings.TrimSpace(text), ""
}

// buildGraphQLBody turns the Body tab's raw text into the JSON payload
// {"query": ..., "variables": ...} that a GraphQL endpoint expects.
func buildGraphQLBody(text string) (string, error) {
	query, variablesJSON := parseGraphQLBody(text)
	payload := map[string]interface{}{"query": query}
	if variablesJSON != "" {
		var vars interface{}
		if err := json.Unmarshal([]byte(variablesJSON), &vars); err != nil {
			return "", fmt.Errorf("invalid GraphQL variables JSON: %w", err)
		}
		payload["variables"] = vars
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
