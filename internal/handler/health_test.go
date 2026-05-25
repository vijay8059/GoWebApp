package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth_Live(t *testing.T) {
	h := NewHealth("test")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	h.Live(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
	if body["version"] != "test" {
		t.Errorf("version = %q, want test", body["version"])
	}
}

func TestHealth_Ready_Toggle(t *testing.T) {
	h := NewHealth("test")

	rr := httptest.NewRecorder()
	h.Ready(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("ready=true got %d, want 200", rr.Code)
	}

	h.SetReady(false)
	rr = httptest.NewRecorder()
	h.Ready(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready=false got %d, want 503", rr.Code)
	}
}

func TestHello(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{"", "hello, world"},
		{"?name=vj", "hello, vj"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		Hello(rr, httptest.NewRequest(http.MethodGet, "/api/v1/hello"+tc.query, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
		var body map[string]string
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		if body["message"] != tc.want {
			t.Errorf("message = %q, want %q", body["message"], tc.want)
		}
	}
}
