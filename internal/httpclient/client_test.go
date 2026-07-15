package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExecute_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok"}`))
	}))
	defer srv.Close()

	resp, err := Execute(&Request{
		Method: "GET",
		URL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"message":"ok"}` {
		t.Errorf("unexpected body: %s", resp.Body)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("missing Content-Type header")
	}
}

func TestExecute_POST_with_body(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("bad json: %v", err)
		}
		if body["name"] != "test" {
			t.Errorf("expected name=test, got %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	resp, err := Execute(&Request{
		Method: "POST",
		URL:    srv.URL,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"name":"test"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if resp.Body != `{"id":1}` {
		t.Errorf("unexpected body: %s", resp.Body)
	}
	if resp.Elapsed <= 0 {
		t.Error("expected positive elapsed time")
	}
	if resp.Size <= 0 {
		t.Error("expected positive size")
	}
}

func TestExecute_headers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("expected Bearer token123, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom: value, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := Execute(&Request{
		Method: "GET",
		URL:    srv.URL,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "value",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExecute_empty_url(t *testing.T) {
	_, err := Execute(&Request{
		Method: "GET",
		URL:    "",
	})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestExecute_timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't write response - will timeout with our 30s client
		// but httptest will close when test ends
	}))
	defer srv.Close()

	// This should succeed or fail gracefully, not hang
	done := make(chan bool, 1)
	go func() {
		resp, err := Execute(&Request{
			Method: "GET",
			URL:    srv.URL,
		})
		if err != nil {
			// Timeout or connection reset is acceptable
			t.Logf("got expected error: %v", err)
		} else if resp != nil {
			t.Logf("got response (unexpected but ok): %d", resp.StatusCode)
		}
		done <- true
	}()

	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("request timed out (hung)")
	}
}

func TestExecute_response_headers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "hello")
		w.Header().Set("X-Num", "42")
		w.Header().Add("X-Multi", "a")
		w.Header().Add("X-Multi", "b")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := Execute(&Request{
		Method: "GET",
		URL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Headers["X-Test"] != "hello" {
		t.Errorf("expected X-Test: hello, got %s", resp.Headers["X-Test"])
	}
	if resp.Headers["X-Num"] != "42" {
		t.Errorf("expected X-Num: 42, got %s", resp.Headers["X-Num"])
	}
	// Multi-value headers should be joined
	if resp.Headers["X-Multi"] != "a, b" {
		t.Errorf("expected X-Multi: 'a, b', got %s", resp.Headers["X-Multi"])
	}
}

func TestExecute_different_methods(t *testing.T) {
	methods := []string{"PUT", "PATCH", "DELETE", "HEAD"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					t.Errorf("expected %s, got %s", method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
				if method != "HEAD" {
					w.Write([]byte(`ok`))
				}
			}))
			defer srv.Close()

			resp, err := Execute(&Request{
				Method: method,
				URL:    srv.URL,
			})
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", method, err)
			}
			if resp.StatusCode != 200 {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestHeaderStr_format(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Custom", "test")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := Execute(&Request{
		Method: "GET",
		URL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(resp.HeaderStr, "Content-Type: text/plain") {
		t.Errorf("expected Content-Type header in HeaderStr, got:\n%s", resp.HeaderStr)
	}
	if !contains(resp.HeaderStr, "X-Custom: test") {
		t.Errorf("expected X-Custom header in HeaderStr, got:\n%s", resp.HeaderStr)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
