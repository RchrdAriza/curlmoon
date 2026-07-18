// Package codegen turns a resolved httpclient.Request into ready-to-run
// snippets in a handful of languages, for curlmoon's "generate code" view.
package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"curlmoon/internal/httpclient"
)

// Lang identifies one of the supported code-generation targets.
type Lang int

const (
	LangCurl Lang = iota
	LangGo
	LangPython
	LangJS
)

// Langs lists every generator in display order, for cycling through in the UI.
var Langs = []Lang{LangCurl, LangGo, LangPython, LangJS}

func (l Lang) String() string {
	switch l {
	case LangCurl:
		return "curl"
	case LangGo:
		return "Go"
	case LangPython:
		return "Python"
	case LangJS:
		return "JavaScript"
	}
	return "?"
}

// Generate renders req as a snippet in the given language.
func Generate(lang Lang, req *httpclient.Request) string {
	switch lang {
	case LangGo:
		return Go(req)
	case LangPython:
		return Python(req)
	case LangJS:
		return JS(req)
	default:
		return Curl(req)
	}
}

// sortedHeaderKeys returns req.Headers' keys in a stable order so generated
// snippets don't reshuffle on every render (map iteration is random).
func sortedHeaderKeys(headers map[string]string) []string {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Curl renders req as a curl command line.
func Curl(req *httpclient.Request) string {
	var b strings.Builder
	b.WriteString("curl -X ")
	b.WriteString(req.Method)
	b.WriteString(" ")
	b.WriteString(shellQuote(req.URL))
	for _, k := range sortedHeaderKeys(req.Headers) {
		b.WriteString(" \\\n  -H ")
		b.WriteString(shellQuote(k + ": " + req.Headers[k]))
	}
	if req.Body != "" {
		b.WriteString(" \\\n  -d ")
		b.WriteString(shellQuote(req.Body))
	}
	return b.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Go renders req as a Go net/http snippet.
func Go(req *httpclient.Request) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	b.WriteString("import (\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n")
	if req.Body != "" {
		b.WriteString("\t\"strings\"\n")
	}
	b.WriteString(")\n\nfunc main() {\n")
	if req.Body != "" {
		fmt.Fprintf(&b, "\tbody := strings.NewReader(%s)\n", strconv.Quote(req.Body))
		fmt.Fprintf(&b, "\treq, err := http.NewRequest(%s, %s, body)\n", strconv.Quote(req.Method), strconv.Quote(req.URL))
	} else {
		fmt.Fprintf(&b, "\treq, err := http.NewRequest(%s, %s, nil)\n", strconv.Quote(req.Method), strconv.Quote(req.URL))
	}
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	for _, k := range sortedHeaderKeys(req.Headers) {
		fmt.Fprintf(&b, "\treq.Header.Set(%s, %s)\n", strconv.Quote(k), strconv.Quote(req.Headers[k]))
	}
	b.WriteString("\n\tresp, err := http.DefaultClient.Do(req)\n")
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n\n")
	b.WriteString("\trespBody, _ := io.ReadAll(resp.Body)\n")
	b.WriteString("\tfmt.Println(resp.Status)\n")
	b.WriteString("\tfmt.Println(string(respBody))\n")
	b.WriteString("}\n")
	return b.String()
}

// Python renders req as a Python requests snippet.
func Python(req *httpclient.Request) string {
	var b strings.Builder
	b.WriteString("import requests\n\n")
	keys := sortedHeaderKeys(req.Headers)
	if len(keys) > 0 {
		b.WriteString("headers = {\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "    %s: %s,\n", pyQuote(k), pyQuote(req.Headers[k]))
		}
		b.WriteString("}\n")
	}
	if req.Body != "" {
		fmt.Fprintf(&b, "data = %s\n", pyQuote(req.Body))
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "response = requests.request(\n    %s,\n    %s,\n", pyQuote(req.Method), pyQuote(req.URL))
	if len(keys) > 0 {
		b.WriteString("    headers=headers,\n")
	}
	if req.Body != "" {
		b.WriteString("    data=data,\n")
	}
	b.WriteString(")\n\n")
	b.WriteString("print(response.status_code)\n")
	b.WriteString("print(response.text)\n")
	return b.String()
}

func pyQuote(s string) string {
	return strconv.Quote(s)
}

// JS renders req as a JavaScript fetch() snippet.
func JS(req *httpclient.Request) string {
	var b strings.Builder
	keys := sortedHeaderKeys(req.Headers)
	fmt.Fprintf(&b, "fetch(%s, {\n", jsQuote(req.URL))
	fmt.Fprintf(&b, "  method: %s,\n", jsQuote(req.Method))
	if len(keys) > 0 {
		b.WriteString("  headers: {\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "    %s: %s,\n", jsQuote(k), jsQuote(req.Headers[k]))
		}
		b.WriteString("  },\n")
	}
	if req.Body != "" {
		fmt.Fprintf(&b, "  body: %s,\n", jsQuote(req.Body))
	}
	b.WriteString("})\n")
	b.WriteString("  .then((res) => res.text().then((text) => console.log(res.status, text)));\n")
	return b.String()
}

func jsQuote(s string) string {
	return strconv.Quote(s)
}
