package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The login sets a cookie as well, for the dashboard. The CLI must take the
// token from the body: the cookie is the credential it deliberately does not
// use.
func TestLoginReadsTheTokenFromTheBodyNotTheCookie(t *testing.T) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			t.Errorf("path = %q, want /api/auth/login", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&body)
		http.SetCookie(w, &http.Cookie{Name: "antenne_session", Value: "cookie-value"})
		w.Write([]byte(`{"user_id":"1","token":"token-abc"}`))
	}))
	defer server.Close()

	token, err := New(server.URL, "").Login(context.Background(), "admin@antenne.local", "hunter2")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token != "token-abc" {
		t.Errorf("token = %q, want the body's token", token)
	}
	if body.Email != "admin@antenne.local" || body.Password != "hunter2" {
		t.Errorf("body = %+v", body)
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

// Antenne serves its dashboard from the same origin, so a wrong path returns 200
// and HTML. Reporting a JSON syntax error there sends the reader hunting for a
// bug in the API instead of a typo in the URL.
func TestHTMLIsReportedAsAWrongURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<!doctype html><title>Antenne</title>"))
	}))
	defer server.Close()

	_, err := New(server.URL, "").Settings(context.Background())
	if err == nil {
		t.Fatal("an HTML body was accepted as settings")
	}
	if !strings.Contains(err.Error(), "Antenne instance") {
		t.Errorf("err = %q, want it to point at the URL", err)
	}
}

// The session travels as a bearer header and never as a cookie: porte reads the
// cookie first, and a cookie-authenticated write without an X-Facile-CSRF
// header is refused while every read keeps working — a failure mode that reads
// as "the tool works, saving is broken".
func TestTheSessionTokenIsSentAsBearerAndNeverAsACookie(t *testing.T) {
	var authorization string
	var cookies int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		cookies = len(r.Cookies())
		w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	api := New(server.URL, "token-xyz")
	if _, err := api.Session(context.Background()); err != nil {
		t.Fatalf("session: %v", err)
	}
	if authorization != "Bearer token-xyz" {
		t.Errorf("Authorization = %q, want the stored token as a bearer", authorization)
	}
	if cookies != 0 {
		t.Errorf("cookies = %d, want none", cookies)
	}

	// A mutating call is the one the CSRF rule would refuse, so it is the one
	// worth checking as well as the read.
	if err := api.TestAlert(context.Background()); err != nil {
		t.Fatalf("test alert: %v", err)
	}
	if authorization != "Bearer token-xyz" || cookies != 0 {
		t.Errorf("mutating call sent Authorization %q and %d cookies", authorization, cookies)
	}
}
