package appleads

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dannolan/apple-ads-cli/internal/config"
)

func TestClientRequestAddsAuthAndOrgContext(t *testing.T) {
	keyPath := writeTestKey(t)
	var sawAuth bool
	var sawOrg bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "expires_in": 3600})
		case "/api/v5/campaigns":
			sawAuth = r.Header.Get("Authorization") == "Bearer test-token"
			sawOrg = r.Header.Get("X-AP-Context") == "orgId=123"
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": 1}}, "pagination": map[string]any{"totalResults": 1}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(config.Credentials{
		OrgID:          123,
		ClientID:       "client",
		TeamID:         "team",
		KeyID:          "key",
		PrivateKeyPath: keyPath,
	})
	client.BaseURL = server.URL + "/api/v5"
	client.TokenURL = server.URL + "/token"

	resp, err := client.Request(RequestOptions{Method: http.MethodGet, Path: "/campaigns"})
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if resp["data"] == nil {
		t.Fatalf("expected data in response: %#v", resp)
	}
	if !sawAuth {
		t.Fatal("expected bearer auth header")
	}
	if !sawOrg {
		t.Fatal("expected X-AP-Context header")
	}
}

func TestClientPaginateCombinesPages(t *testing.T) {
	keyPath := writeTestKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "expires_in": 3600})
		case "/api/v5/campaigns":
			offset := r.URL.Query().Get("offset")
			if offset == "0" {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": 1}}, "pagination": map[string]any{"totalResults": 2}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": 2}}, "pagination": map[string]any{"totalResults": 2}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(config.Credentials{OrgID: 123, ClientID: "client", TeamID: "team", KeyID: "key", PrivateKeyPath: keyPath})
	client.BaseURL = server.URL + "/api/v5"
	client.TokenURL = server.URL + "/token"

	items, err := client.Paginate("/campaigns", nil)
	if err != nil {
		t.Fatalf("Paginate() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func writeTestKey(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "key.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
