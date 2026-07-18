package tui

import (
	"encoding/base64"
	"strings"
)

// authTypes lists the auth helpers curlmoon understands, cycled via the same
// key as CycleBodyType (Ctrl+Y by default) while the Auth tab is active. The
// Auth content editor's meaning depends on which of these is selected — see
// authPlaceholder and applyAuth.
var authTypes = []string{"None", "Basic", "Bearer", "API Key", "OAuth2"}

const (
	authNone = iota
	authBasic
	authBearer
	authAPIKey
	authOAuth2
)

// authPlaceholder seeds the content editor when the user switches to a given
// auth type on an empty Auth tab, so they see the expected shape instead of
// a blank box.
func authPlaceholder(authType int) string {
	switch authType {
	case authBasic:
		return "Username: \nPassword: "
	case authAPIKey:
		return "Key: X-API-Key\nValue: "
	default:
		return ""
	}
}

// applyAuth injects the Authorization (or custom) header derived from
// authType/authText into h. Bearer and OAuth2 take authText as a raw token
// verbatim — paste it straight in, no "Key: Value" wrapping needed. Basic and
// API Key still need two fields, so they keep the per-line format used by
// headers/params.
func applyAuth(h map[string]string, authType int, authText string) {
	switch authType {
	case authBasic:
		fields := kvToMap(parseKV(authText))
		user := fieldGet(fields, "Username")
		pass := fieldGet(fields, "Password")
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		h["Authorization"] = "Basic " + token
	case authBearer, authOAuth2:
		token := strings.TrimSpace(authText)
		if token == "" {
			return
		}
		h["Authorization"] = "Bearer " + token
	case authAPIKey:
		fields := kvToMap(parseKV(authText))
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
