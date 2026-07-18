package codegen

import (
	"strings"
	"testing"

	"curlmoon/internal/httpclient"
)

func sampleRequest() *httpclient.Request {
	return &httpclient.Request{
		Method:  "POST",
		URL:     "https://api.example.com/items",
		Headers: map[string]string{"Content-Type": "application/json", "X-Api-Key": "secret"},
		Body:    `{"name":"widget"}`,
	}
}

func TestCurl(t *testing.T) {
	out := Curl(sampleRequest())
	for _, want := range []string{"curl -X POST", "https://api.example.com/items", "-H 'Content-Type: application/json'", "-H 'X-Api-Key: secret'", `-d '{"name":"widget"}'`} {
		if !strings.Contains(out, want) {
			t.Errorf("curl output missing %q:\n%s", want, out)
		}
	}
}

func TestCurl_ShellEscapesSingleQuotes(t *testing.T) {
	req := sampleRequest()
	req.Body = `it's a test`
	out := Curl(req)
	if !strings.Contains(out, `'it'\''s a test'`) {
		t.Errorf("expected escaped single quote, got:\n%s", out)
	}
}

func TestGo(t *testing.T) {
	out := Go(sampleRequest())
	for _, want := range []string{`http.NewRequest("POST", "https://api.example.com/items"`, `req.Header.Set("X-Api-Key", "secret")`, "http.DefaultClient.Do(req)"} {
		if !strings.Contains(out, want) {
			t.Errorf("go output missing %q:\n%s", want, out)
		}
	}
}

func TestPython(t *testing.T) {
	out := Python(sampleRequest())
	for _, want := range []string{"import requests", `"POST"`, `"https://api.example.com/items"`, "headers=headers", "data=data"} {
		if !strings.Contains(out, want) {
			t.Errorf("python output missing %q:\n%s", want, out)
		}
	}
}

func TestJS(t *testing.T) {
	out := JS(sampleRequest())
	for _, want := range []string{`fetch("https://api.example.com/items"`, `method: "POST"`, "headers:", "body:"} {
		if !strings.Contains(out, want) {
			t.Errorf("js output missing %q:\n%s", want, out)
		}
	}
}

func TestGenerate_DispatchesByLang(t *testing.T) {
	req := sampleRequest()
	cases := map[Lang]string{
		LangCurl:   "curl",
		LangGo:     "package main",
		LangPython: "import requests",
		LangJS:     "fetch(",
	}
	for lang, want := range cases {
		out := Generate(lang, req)
		if !strings.Contains(out, want) {
			t.Errorf("Generate(%v) missing %q:\n%s", lang, want, out)
		}
	}
}

func TestNoBody_OmitsBodyFields(t *testing.T) {
	req := sampleRequest()
	req.Body = ""
	if strings.Contains(Curl(req), "-d") {
		t.Error("curl should not include -d when body is empty")
	}
	if strings.Contains(Python(req), "data=data") {
		t.Error("python should not include data= when body is empty")
	}
	if strings.Contains(JS(req), "body:") {
		t.Error("js should not include body: when body is empty")
	}
}

func TestLangString(t *testing.T) {
	if LangCurl.String() != "curl" || LangGo.String() != "Go" || LangPython.String() != "Python" || LangJS.String() != "JavaScript" {
		t.Error("unexpected Lang.String() values")
	}
}
