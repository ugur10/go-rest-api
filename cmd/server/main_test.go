package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractID(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "valid", path: "/5", want: "5"},
		{name: "nested slash", path: "/5/more", wantErr: true},
		{name: "missing leading slash", path: "5", wantErr: true},
		{name: "empty", path: "/", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractID(tc.path)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestReadBookPayload(t *testing.T) {
	body := bytes.NewBufferString(`{"title":"Go","author":"Someone","isbn":"1","publishedYear":2020}`)
	req := httptest.NewRequest(http.MethodPost, "/api/books", body)
	req.Header.Set("Content-Type", "application/json")

	payload, err := readBookPayload(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.Title != "Go" || payload.Author != "Someone" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestReadBookPayload_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")

	if _, err := readBookPayload(req); err == nil {
		t.Fatalf("expected error")
	}
}

func TestContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rr := httptest.NewRecorder()
	if ok := contextDone(rr, ctx); !ok {
		t.Fatalf("expected contextDone to return true")
	}

	if rr.Code != http.StatusRequestTimeout {
		t.Fatalf("expected %d, got %d", http.StatusRequestTimeout, rr.Code)
	}
}
