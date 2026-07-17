package tui

import (
	"encoding/base64"
	"strings"
)

// defaultAuthText seeds the Auth tab so users can see the expected format.
const defaultAuthText = "Type: None"

// authTypes lists the auth helpers curlmoon understands via the "Type: ..."
// line at the top of the Auth tab's text.
var authTypes = []string{"None", "Basic", "Bearer", "API Key", "OAuth2"}

// applyAuth parses authText (the same "Key: Value" per-line format used by
// the headers/params editors) and injects the resulting Authorization (or
// custom) header into h. authText's first line selects the auth type, e.g.:
//
//	Type: Basic
//	Username: alice
//	Password: secret
//
//	Type: Bearer
//	Token: {{api_token}}
//
//	Type: API Key
//	Key: X-API-Key
//	Value: {{api_key}}
//
//	Type: OAuth2
//	Token: {{access_token}}
func applyAuth(h map[string]string, authText string) {
	fields := kvToMap(parseKV(authText))
	authType := fieldGet(fields, "Type")

	switch strings.ToLower(authType) {
	case "", "none":
		return
	case "basic":
		user := fieldGet(fields, "Username")
		pass := fieldGet(fields, "Password")
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		h["Authorization"] = "Basic " + token
	case "bearer":
		h["Authorization"] = "Bearer " + fieldGet(fields, "Token")
	case "oauth2":
		h["Authorization"] = "Bearer " + fieldGet(fields, "Token")
	case "api key":
		key := fieldGet(fields, "Key")
		if key == "" {
			key = "X-API-Key"
		}
		h[key] = fieldGet(fields, "Value")
	}
}

// fieldGet looks up a key in fields case-insensitively.
func fieldGet(fields map[string]string, key string) string {
	for k, v := range fields {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}
