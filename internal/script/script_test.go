package script

import (
	"testing"

	"curlmoon/internal/httpclient"
)

func TestRunPreRequest_Empty(t *testing.T) {
	env := map[string]string{}
	res := RunPreRequest("", env)
	if res.PreRequestErr != "" {
		t.Errorf("expected no error, got %q", res.PreRequestErr)
	}
}

func TestRunPreRequest_SetsEnvironmentVariable(t *testing.T) {
	env := map[string]string{"existing": "1"}
	res := RunPreRequest(`pm.environment.set("token", "abc123");`, env)
	if res.PreRequestErr != "" {
		t.Fatalf("unexpected error: %s", res.PreRequestErr)
	}
	if env["token"] != "abc123" {
		t.Errorf("expected env[token]=abc123, got %v", env)
	}
	if env["existing"] != "1" {
		t.Errorf("existing env vars should survive, got %v", env)
	}
}

func TestRunPreRequest_ReadsEnvironmentVariable(t *testing.T) {
	env := map[string]string{"base": "https://api.example.com"}
	res := RunPreRequest(`pm.environment.set("copy", pm.environment.get("base"));`, env)
	if res.PreRequestErr != "" {
		t.Fatalf("unexpected error: %s", res.PreRequestErr)
	}
	if env["copy"] != "https://api.example.com" {
		t.Errorf("expected copy to mirror base, got %v", env)
	}
}

func TestRunPreRequest_ScriptError(t *testing.T) {
	res := RunPreRequest(`this is not valid javascript(((`, map[string]string{})
	if res.PreRequestErr == "" {
		t.Error("expected a script error")
	}
}

func TestRunTest_PassingAssertion(t *testing.T) {
	resp := &httpclient.Response{StatusCode: 200}
	res := RunTest(`pm.test("status is 200", () => pm.response.code === 200);`, map[string]string{}, &httpclient.Request{}, resp)
	if len(res.Tests) != 1 || !res.Tests[0].Passed {
		t.Fatalf("expected 1 passing test, got %+v", res.Tests)
	}
	if res.Tests[0].Name != "status is 200" {
		t.Errorf("unexpected test name %q", res.Tests[0].Name)
	}
}

func TestRunTest_FailingAssertion(t *testing.T) {
	resp := &httpclient.Response{StatusCode: 500}
	res := RunTest(`pm.test("status is 200", () => pm.response.code === 200);`, map[string]string{}, &httpclient.Request{}, resp)
	if len(res.Tests) != 1 || res.Tests[0].Passed {
		t.Fatalf("expected 1 failing test, got %+v", res.Tests)
	}
}

func TestRunTest_ThrowingAssertion(t *testing.T) {
	resp := &httpclient.Response{StatusCode: 200}
	res := RunTest(`pm.test("throws", () => { throw new Error("boom"); });`, map[string]string{}, &httpclient.Request{}, resp)
	if len(res.Tests) != 1 || res.Tests[0].Passed {
		t.Fatalf("expected 1 failing test, got %+v", res.Tests)
	}
	if res.Tests[0].Err == "" {
		t.Error("expected error message on failing test")
	}
}

func TestRunTest_MultipleTests(t *testing.T) {
	resp := &httpclient.Response{StatusCode: 200, Body: `{"ok":true}`}
	script := `
		pm.test("status is 200", () => pm.response.code === 200);
		pm.test("body has ok", () => pm.response.json().ok === true);
		pm.test("this fails", () => false);
	`
	res := RunTest(script, map[string]string{}, &httpclient.Request{}, resp)
	if len(res.Tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(res.Tests))
	}
	passed := 0
	for _, tr := range res.Tests {
		if tr.Passed {
			passed++
		}
	}
	if passed != 2 {
		t.Errorf("expected 2 passing tests, got %d", passed)
	}
}

func TestRunTest_ResponseHeadersAndText(t *testing.T) {
	resp := &httpclient.Response{StatusCode: 200, Body: "hello", Headers: map[string]string{"X-Custom": "yes"}}
	res := RunTest(`pm.test("header check", () => pm.response.headers.get("X-Custom") === "yes" && pm.response.text() === "hello");`, map[string]string{}, &httpclient.Request{}, resp)
	if len(res.Tests) != 1 || !res.Tests[0].Passed {
		t.Fatalf("expected passing test, got %+v", res.Tests)
	}
}

func TestRunTest_RequestHeaderAccess(t *testing.T) {
	req := &httpclient.Request{Headers: map[string]string{"Authorization": "Bearer tok"}}
	resp := &httpclient.Response{StatusCode: 200}
	res := RunTest(`pm.test("auth header", () => pm.request.headers.get("Authorization") === "Bearer tok");`, map[string]string{}, req, resp)
	if len(res.Tests) != 1 || !res.Tests[0].Passed {
		t.Fatalf("expected passing test, got %+v", res.Tests)
	}
}

func TestRunTest_Empty(t *testing.T) {
	res := RunTest("", map[string]string{}, &httpclient.Request{}, &httpclient.Response{})
	if len(res.Tests) != 0 || res.TestErr != "" {
		t.Errorf("expected no-op result, got %+v", res)
	}
}
