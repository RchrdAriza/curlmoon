package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegration_FullRequestResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","data":[1,2,3]}`))
	}))
	defer srv.Close()

	a := NewApp()
	a.urlValue = srv.URL
	a.methodIndex = 0

	resp, err := a.doRequest()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	a.HandleResponse(resp, err)

	if a.response == nil {
		t.Fatal("expected response to be set")
	}
	if a.response.StatusCode != 200 {
		t.Errorf("expected 200, got %d", a.response.StatusCode)
	}
	if !strings.Contains(a.response.Body, "status") {
		t.Errorf("expected response body, got: %s", a.response.Body)
	}
	if len(a.response.Headers) == 0 {
		t.Error("expected response headers")
	}
	if a.response.Size <= 0 {
		t.Error("expected positive size")
	}
	if a.response.Elapsed <= 0 {
		t.Error("expected positive elapsed time")
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	a := NewApp()
	a.urlValue = "http://nonexistent.invalid/api"
	a.methodIndex = 0

	resp, err := a.doRequest()
	a.HandleResponse(resp, err)

	if a.respErr == nil {
		t.Error("expected error for invalid URL")
	}
	if a.showResp {
		t.Error("expected showResp=false on error")
	}
	if !strings.Contains(a.statusMsg, "Error") {
		t.Error("expected statusMsg to show error")
	}
}

func TestIntegration_SuccessResponseView(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"success"}`))
	}))
	defer srv.Close()

	a := NewApp()
	a.urlValue = srv.URL

	resp, err := a.doRequest()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	a.HandleResponse(resp, err)

	if !strings.Contains(a.statusMsg, "200") {
		t.Errorf("expected status code 200 in status message, got %q", a.statusMsg)
	}
	if !strings.Contains(a.response.Body, "success") {
		t.Errorf("expected response body to contain success, got %q", a.response.Body)
	}
}
