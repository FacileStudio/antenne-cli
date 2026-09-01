package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apikeys" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if app := r.URL.Query().Get("app"); app != "" && app != "webapp" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(ListKeysResponse{
			Keys: []Key{
				{
					ID:             1,
					App:            "webapp",
					Kind:           "secret",
					Prefix:         "antenne_webapp_abc",
					AllowedOrigins: []string{},
					DailyQuota:     0,
					UsedToday:      10,
					CreatedAt:      "2026-09-01T15:00:00Z",
				},
			},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	c := New(server.URL, "test-token")
	keys, err := c.ListKeys(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].App != "webapp" || keys[0].UsedToday != 10 {
		t.Fatalf("unexpected key content: %+v", keys[0])
	}

	keysFiltered, err := c.ListKeys(context.Background(), "webapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keysFiltered) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keysFiltered))
	}
}

func TestCreateKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apikeys" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req CreateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(CreateKeyResponse{
			Key: Key{
				ID:             2,
				App:            req.App,
				Kind:           req.Kind,
				Prefix:         "antenne_pub_webapp_xyz",
				AllowedOrigins: req.AllowedOrigins,
				DailyQuota:     req.DailyQuota,
				CreatedAt:      "2026-09-01T15:00:00Z",
			},
			Token: "antenne_pub_webapp_xyz123456",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	c := New(server.URL, "test-token")
	resp, err := c.CreateKey(context.Background(), CreateKeyRequest{
		App:            "webapp",
		Kind:           "public",
		AllowedOrigins: []string{"https://example.com"},
		DailyQuota:     5000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "antenne_pub_webapp_xyz123456" {
		t.Fatalf("expected token, got %s", resp.Token)
	}
	if resp.Key.App != "webapp" || resp.Key.DailyQuota != 5000 {
		t.Fatalf("unexpected key payload: %+v", resp.Key)
	}
}

func TestRevokeKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apikeys/2" || r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := New(server.URL, "test-token")
	err := c.RevokeKey(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
