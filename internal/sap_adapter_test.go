package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSAPAuth_FetchCSRFToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") != "Fetch" {
			t.Errorf("expected X-CSRF-Token: Fetch header")
		}
		w.Header().Set("X-CSRF-Token", "abc-csrf-123")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	auth := NewSAPAuth(srv.URL, SAPAuthConfig{AuthType: "basic", Username: "user", Password: "pass"}, srv.Client())
	token, err := auth.FetchCSRFToken(context.Background())
	if err != nil {
		t.Fatalf("FetchCSRFToken failed: %v", err)
	}
	if token != "abc-csrf-123" {
		t.Errorf("expected abc-csrf-123, got %q", token)
	}
	if auth.CSRFToken() != "abc-csrf-123" {
		t.Errorf("cached token mismatch")
	}
}

func TestSAPAuth_FetchCSRFToken_Missing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	auth := NewSAPAuth(srv.URL, SAPAuthConfig{AuthType: "basic", Username: "u", Password: "p"}, srv.Client())
	_, err := auth.FetchCSRFToken(context.Background())
	if err == nil {
		t.Fatal("expected error when no CSRF token in response")
	}
}

func TestSAPAuth_BasicAuth(t *testing.T) {
	auth := NewSAPAuth("http://example.com", SAPAuthConfig{
		AuthType: "basic",
		Username: "admin",
		Password: "secret",
	}, nil)
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	if err := auth.AuthHeader()(req); err != nil {
		t.Fatalf("AuthHeader failed: %v", err)
	}
	if req.Header.Get("Authorization") == "" {
		t.Error("expected Authorization header")
	}
}

func TestSAPAuth_APIKeyAuth(t *testing.T) {
	auth := NewSAPAuth("http://example.com", SAPAuthConfig{
		AuthType: "apikey",
		APIKey:   "my-key-123",
	}, nil)
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	if err := auth.AuthHeader()(req); err != nil {
		t.Fatalf("AuthHeader failed: %v", err)
	}
	if req.Header.Get("APIKey") != "my-key-123" {
		t.Errorf("expected APIKey header, got %q", req.Header.Get("APIKey"))
	}
}

func TestSAPAuth_OAuth2(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST to token endpoint")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "oauth-token-xyz",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenSrv.Close()

	auth := NewSAPAuth("http://example.com", SAPAuthConfig{
		AuthType:     "oauth2",
		ClientID:     "cid",
		ClientSecret: "csecret",
		TokenURL:     tokenSrv.URL,
	}, tokenSrv.Client())

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	if err := auth.AuthHeader()(req); err != nil {
		t.Fatalf("AuthHeader failed: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer oauth-token-xyz" {
		t.Errorf("expected Bearer oauth-token-xyz, got %q", req.Header.Get("Authorization"))
	}

	// Second call should use cached token
	req2, _ := http.NewRequest("GET", "http://example.com/test2", nil)
	if err := auth.AuthHeader()(req2); err != nil {
		t.Fatalf("AuthHeader (cached) failed: %v", err)
	}
	if req2.Header.Get("Authorization") != "Bearer oauth-token-xyz" {
		t.Errorf("cached token mismatch")
	}
}

func TestSAPAuth_UnsupportedType(t *testing.T) {
	auth := NewSAPAuth("http://example.com", SAPAuthConfig{AuthType: "kerberos"}, nil)
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	if err := auth.AuthHeader()(req); err == nil {
		t.Error("expected error for unsupported auth type")
	}
}

func TestSAPAuth_FetchCSRFToken_OAuth2_NoDeadlock(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "csrf-oauth-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenSrv.Close()

	csrfSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer csrf-oauth-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-CSRF-Token", "csrf-from-oauth")
		w.WriteHeader(http.StatusOK)
	}))
	defer csrfSrv.Close()

	auth := NewSAPAuth(csrfSrv.URL, SAPAuthConfig{
		AuthType:     "oauth2",
		ClientID:     "cid",
		ClientSecret: "csecret",
		TokenURL:     tokenSrv.URL,
	}, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		token, err := auth.FetchCSRFToken(context.Background())
		if err != nil {
			t.Errorf("FetchCSRFToken with OAuth2 failed: %v", err)
			return
		}
		if token != "csrf-from-oauth" {
			t.Errorf("expected csrf-from-oauth, got %q", token)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("FetchCSRFToken deadlocked with OAuth2 auth")
	}
}

func TestSAPAdapter_ConnectAndCRUD(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/Orders('1')":
			json.NewEncoder(w).Encode(map[string]any{"ID": "1", "Status": "Open"})
		case r.Method == http.MethodGet && r.URL.Path == "/Orders":
			json.NewEncoder(w).Encode(map[string]any{
				"value":         []map[string]any{{"ID": "1"}},
				"@odata.count":  1,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/Orders":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"ID": "2"})
		case r.Method == http.MethodPatch && r.URL.Path == "/Orders('1')":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/Orders('1')":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	adapter := NewSAPAdapter()
	err := adapter.Connect(context.Background(), ProviderConfig{
		BaseURL:  srv.URL,
		AuthType: "basic",
		Username: "test",
		Password: "test",
	})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// ReadEntity
	entity, err := adapter.ReadEntity(context.Background(), "Orders", "'1'")
	if err != nil {
		t.Fatalf("ReadEntity: %v", err)
	}
	if entity["ID"] != "1" {
		t.Errorf("read: unexpected ID %v", entity["ID"])
	}

	// QueryEntities
	qr, err := adapter.QueryEntities(context.Background(), "Orders", QueryOptions{})
	if err != nil {
		t.Fatalf("QueryEntities: %v", err)
	}
	if len(qr.Results) != 1 {
		t.Errorf("query: expected 1 result, got %d", len(qr.Results))
	}

	// CreateEntity
	created, err := adapter.CreateEntity(context.Background(), "Orders", map[string]any{"Name": "New"})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	if created["ID"] != "2" {
		t.Errorf("create: unexpected ID %v", created["ID"])
	}

	// UpdateEntity
	if err := adapter.UpdateEntity(context.Background(), "Orders", "'1'", map[string]any{"Status": "Done"}); err != nil {
		t.Fatalf("UpdateEntity: %v", err)
	}

	// DeleteEntity
	if err := adapter.DeleteEntity(context.Background(), "Orders", "'1'"); err != nil {
		t.Fatalf("DeleteEntity: %v", err)
	}

	// Close
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
