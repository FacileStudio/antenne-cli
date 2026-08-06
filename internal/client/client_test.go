package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginReturnsTheSessionCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "token-abc"})
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	token, err := New(server.URL, "").Login(context.Background(), "hunter2")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token != "token-abc" {
		t.Errorf("token = %q, want the cookie value", token)
	}
}

// An instance with no admin password answers 200 and sets no cookie, because
// every caller is already the admin. That is a success with an empty token, not
// a failure.
func TestLoginAgainstAnOpenInstanceReturnsNoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	token, err := New(server.URL, "").Login(context.Background(), "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

// The suite error envelope has to survive as a code the caller can branch on,
// or every failure collapses to "HTTP 401" with no explanation.
func TestTheErrorEnvelopeIsDecoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"code":"unauthenticated","message":"invalid password"}}`))
	}))
	defer server.Close()

	_, err := New(server.URL, "").Session(context.Background())
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %T, want *Error", err)
	}
	if apiErr.Code != "unauthenticated" || apiErr.Message != "invalid password" {
		t.Errorf("err = %+v", apiErr)
	}
	if !apiErr.Unauthenticated() {
		t.Error("Unauthenticated() = false on a 401")
	}
}

// Nook serves its dashboard from the same origin, so a wrong path returns 200
// and HTML. Reporting a JSON syntax error there sends the reader hunting for a
// bug in the API instead of a typo in the URL.
func TestHTMLIsReportedAsAWrongURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<!doctype html><title>Nook</title>"))
	}))
	defer server.Close()

	_, err := New(server.URL, "").Settings(context.Background())
	if err == nil {
		t.Fatal("an HTML body was accepted as settings")
	}
	if !strings.Contains(err.Error(), "Nook instance") {
		t.Errorf("err = %q, want it to point at the URL", err)
	}
}

// The session travels as a cookie; the API reads nothing else.
func TestTheSessionTokenIsSentAsACookie(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(SessionCookie); err == nil {
			seen = cookie.Value
		}
		w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	if _, err := New(server.URL, "token-xyz").Session(context.Background()); err != nil {
		t.Fatalf("session: %v", err)
	}
	if seen != "token-xyz" {
		t.Errorf("cookie = %q, want the stored token", seen)
	}
}
