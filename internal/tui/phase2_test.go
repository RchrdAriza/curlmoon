package tui

import (
	"strings"
	"testing"
)

func TestAppBodyTypeDefault(t *testing.T) {
	a := NewApp()
	if a.bodyType != 0 {
		t.Errorf("expected bodyType=0 (none), got %d", a.bodyType)
	}
}

func TestAppSendWithHeaders(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://httpbin.org/post"
	a.methodIndex = 1
	a.headersText = "X-Custom: test123"
	a.bodyType = 1
	a.bodyText = `{"data":"test"}`

	a.StartSending()
	if !a.sending {
		t.Error("expected sending=true")
	}
	if !strings.Contains(a.statusMsg, "Sending") {
		t.Error("expected statusMsg to show sending")
	}

	resp, err := a.doRequest()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	a.HandleResponse(resp, err)

	if a.response == nil {
		t.Fatal("expected response")
	}
	if a.response.StatusCode != 200 {
		t.Errorf("expected 200, got %d", a.response.StatusCode)
	}
	if a.sending {
		t.Error("expected sending=false after HandleResponse")
	}
}

func TestAppParamsUpdateURL(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://httpbin.org/get"
	a.paramsText = "q: search"

	resp, err := a.doRequest()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	a.HandleResponse(resp, err)

	if a.response == nil {
		t.Fatal("expected response")
	}
	if !strings.Contains(a.response.Body, "search") {
		t.Errorf("expected 'search' in response args, got:\n%s", a.response.Body)
	}
}
