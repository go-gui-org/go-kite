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
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"identifier":"alice"`) {
			t.Fatalf("missing identifier in payload: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"handle":"h","email":"e","accessJwt":"a","refreshJwt":"r"}`))
	}))
	defer server.Close()

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	baseURL, _ := url.Parse(server.URL)
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
		_, _ = w.Write([]byte(`{"feed":[]}`))
	}))
	defer server.Close()

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	baseURL, _ := url.Parse(server.URL)
	httpClient = &http.Client{Transport: rewriteTransport{base: baseURL, rt: http.DefaultTransport}}

	_, err := getTimeline(BSkySession{AccessJWT: "token"})
	if err != nil {
		t.Fatalf("getTimeline error: %v", err)
	}
}
