package oauth

import (
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"9router/proxy/internal/db"
)

func TestHandleFreebuffSessionStatus_MethodNotAllowed(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	handler := NewOAuthHandler(db.NewRepo(database))

	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/session", nil)
	rec := httptest.NewRecorder()

	handler.HandleFreebuffSessionStatus(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleFreebuffSessionStatus_NoConnection(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	handler := NewOAuthHandler(db.NewRepo(database))

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/freebuff/session", nil)
	rec := httptest.NewRecorder()

	handler.HandleFreebuffSessionStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if res["status"] != "none" {
		t.Fatalf("expected status=none, got %v", res["status"])
	}
}

func TestHandleFreebuffSessionStatus_ActiveSession(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	handler := NewOAuthHandler(db.NewRepo(database))

	var authHeader, uaHeader, acceptHeader string
	var upstreamCalls int64

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&upstreamCalls, 1)
		authHeader = r.Header.Get("Authorization")
		uaHeader = r.Header.Get("User-Agent")
		acceptHeader = r.Header.Get("Accept")

		if r.URL.Path != "/api/v1/freebuff/session" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "active",
			"model": "z-ai/glm-5.3-flash",
			"instanceId": "inst-12345",
			"expiresAt": "2026-09-17T01:00:00Z",
			"freebucks": {
				"balance": 20
			}
		}`))
	}))
	defer mockUpstream.Close()

	origBaseURL := freebuffAPIBaseURL
	freebuffAPIBaseURL = mockUpstream.URL
	defer func() { freebuffAPIBaseURL = origBaseURL }()

	// Seed active freebuff connection
	connData, _ := json.Marshal(map[string]any{"authToken": "fb-secret-token-123"})
	_, err := database.Exec(`
		INSERT INTO providerConnections (id, provider, authType, name, isActive, data, createdAt, updatedAt)
		VALUES ('fb-conn-1', 'freebuff', 'oauth', 'Test FB', 1, ?, '2026-09-17T00:00:00Z', '2026-09-17T00:00:00Z')
	`, string(connData))
	if err != nil {
		t.Fatalf("failed to seed connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/freebuff/session?connectionId=fb-conn-1", nil)
	rec := httptest.NewRecorder()

	handler.HandleFreebuffSessionStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if authHeader != "Bearer fb-secret-token-123" {
		t.Errorf("expected Authorization Bearer fb-secret-token-123, got %q", authHeader)
	}
	if uaHeader != "codebuff-cli/0.0.138" {
		t.Errorf("expected User-Agent codebuff-cli/0.0.138, got %q", uaHeader)
	}
	if acceptHeader != "application/json" {
		t.Errorf("expected Accept application/json, got %q", acceptHeader)
	}

	var res struct {
		Status       string         `json:"status"`
		CurrentModel string         `json:"currentModel"`
		InstanceID   string         `json:"instanceId"`
		ExpiresAt    string         `json:"expiresAt"`
		Freebucks    map[string]any `json:"freebucks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if res.Status != "active" {
		t.Errorf("expected status active, got %q", res.Status)
	}
	if res.CurrentModel != "z-ai/glm-5.3-flash" {
		t.Errorf("expected currentModel z-ai/glm-5.3-flash, got %q", res.CurrentModel)
	}
	if res.InstanceID != "inst-12345" {
		t.Errorf("expected instanceId inst-12345, got %q", res.InstanceID)
	}
	if res.ExpiresAt != "2026-09-17T01:00:00Z" {
		t.Errorf("expected expiresAt 2026-09-17T01:00:00Z, got %q", res.ExpiresAt)
	}
	if res.Freebucks == nil || res.Freebucks["balance"] != float64(20) {
		t.Errorf("expected freebucks.balance=20, got %v", res.Freebucks)
	}
}

func TestHandleFreebuffSessionStatus_Unauthorized(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	handler := NewOAuthHandler(db.NewRepo(database))

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
	}))
	defer mockUpstream.Close()

	origBaseURL := freebuffAPIBaseURL
	freebuffAPIBaseURL = mockUpstream.URL
	defer func() { freebuffAPIBaseURL = origBaseURL }()

	connData, _ := json.Marshal(map[string]any{"accessToken": "expired-token"})
	_, err := database.Exec(`
		INSERT INTO providerConnections (id, provider, authType, name, isActive, data, createdAt, updatedAt)
		VALUES ('fb-conn-2', 'freebuff', 'oauth', 'Test FB 2', 1, ?, '2026-09-17T00:00:00Z', '2026-09-17T00:00:00Z')
	`, string(connData))
	if err != nil {
		t.Fatalf("failed to seed connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/freebuff/session", nil)
	rec := httptest.NewRecorder()

	handler.HandleFreebuffSessionStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if res["status"] != "unauthorized" {
		t.Errorf("expected status=unauthorized, got %v", res["status"])
	}
}
