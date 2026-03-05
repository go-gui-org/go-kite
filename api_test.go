package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type rewriteTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.base.Scheme
	req.URL.Host = t.base.Host
	return t.rt.RoundTrip(req)
}

func TestCreateSessionSendsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.createSession" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), `"identifier":"alice"`) {
			t.Fatalf("missing identifier in payload: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"handle":"h","email":"e","accessJwt":"a","refreshJwt":"r"}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	httpClient = &http.Client{Transport: rewriteTransport{base: baseURL, rt: http.DefaultTransport}}

	s, err := createSession("alice", "pw")
	if err != nil {
		t.Fatalf("createSession error: %v", err)
	}
	if s.AccessJWT != "a" || s.RefreshJWT != "r" {
		t.Fatalf("unexpected session: %+v", s)
	}
}

func TestGetTimelineSendsAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		if _, err := w.Write([]byte(`{"feed":[]}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	httpClient = &http.Client{Transport: rewriteTransport{base: baseURL, rt: http.DefaultTransport}}

	_, err = getTimeline(BSkySession{AccessJWT: "token"})
	if err != nil {
		t.Fatalf("getTimeline error: %v", err)
	}
}
