package oauth

import (
	"database/sql"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"9router/proxy/internal/db"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	tmpFile, err := os.CreateTemp("", "test_freebuff_*.sqlite")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	database, err := db.OpenDatabase(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("OpenDatabase failed: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS providerConnections (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		authType TEXT NOT NULL,
		name TEXT,
		email TEXT,
		priority INTEGER,
		isActive INTEGER DEFAULT 1,
		data TEXT NOT NULL,
		createdAt TEXT,
		updatedAt TEXT
	);`
	if _, err := database.Exec(schema); err != nil {
		database.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("exec schema failed: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.Remove(tmpFile.Name())
	}
	return database, cleanup
}

func TestHandleFreebuffInitiate_Success(t *testing.T) {
	handler := NewOAuthHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/initiate", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleFreebuffInitiate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	loginURL, _ := res["loginUrl"].(string)
	if !strings.HasPrefix(loginURL, "https://freebuff.com/login?auth_code=") {
		t.Errorf("unexpected loginUrl: %v", loginURL)
	}
	authCode, _ := res["authCode"].(string)
	if len(authCode) == 0 {
		t.Errorf("empty authCode")
	}
	fpID, _ := res["fingerprintId"].(string)
	if len(fpID) != 36 {
		t.Errorf("unexpected fingerprintId length: %v", fpID)
	}
	fpHash, _ := res["fingerprintHash"].(string)
	if len(fpHash) != 64 {
		t.Errorf("unexpected fingerprintHash length: %v", fpHash)
	}
	if res["expiresAt"] == "" {
		t.Errorf("empty expiresAt")
	}
}


func TestHandleFreebuffInitiate_MethodNotAllowed(t *testing.T) {
	handler := NewOAuthHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/freebuff/initiate", nil)
	rec := httptest.NewRecorder()

	handler.HandleFreebuffInitiate(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}


func TestHandleFreebuffPoll_MissingFields(t *testing.T) {
	handler := NewOAuthHandler(nil)

	// Missing fingerprintHash
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/poll", strings.NewReader(`{"fingerprintId":"fp-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleFreebuffPoll(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	// Empty body
	req2 := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/poll", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.HandleFreebuffPoll(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec2.Code)
	}
}

func TestHandleFreebuffPoll_Pending(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/cli/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "pending"}`))
	}))
	defer mockServer.Close()

	oldURL := freebuffAuthBaseURL
	freebuffAuthBaseURL = mockServer.URL
	defer func() { freebuffAuthBaseURL = oldURL }()

	handler := NewOAuthHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/poll", strings.NewReader(`{"fingerprintId":"fp-1","fingerprintHash":"hash-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleFreebuffPoll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if res["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", res["status"])
	}
	if res["connectionId"] != nil {
		t.Errorf("expected connectionId to be empty for pending, got %v", res["connectionId"])
	}
}

func TestHandleFreebuffPoll_Expired(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "expired"}`))
	}))
	defer mockServer.Close()

	oldURL := freebuffAuthBaseURL
	freebuffAuthBaseURL = mockServer.URL
	defer func() { freebuffAuthBaseURL = oldURL }()

	handler := NewOAuthHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/poll", strings.NewReader(`{"fingerprintId":"fp-1","fingerprintHash":"hash-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleFreebuffPoll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if res["status"] != "expired" {
		t.Errorf("expected status 'expired', got %v", res["status"])
	}
}

func TestHandleFreebuffPoll_Authorized_CreatesConnection(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	repo := db.NewRepo(database)

	testAuthToken := "fb_token_live_123456789abcdef"
	expectedConnID := "fb-" + shortHash(testAuthToken)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/cli/status" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "authorized",
			"authToken": "` + testAuthToken + `",
			"email": "user@freebuff.com",
			"name": "Freebuff Master"
		}`))
	}))
	defer mockServer.Close()

	oldURL := freebuffAuthBaseURL
	freebuffAuthBaseURL = mockServer.URL
	defer func() { freebuffAuthBaseURL = oldURL }()

	handler := NewOAuthHandler(repo)
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/poll", strings.NewReader(`{
		"fingerprintId": "fp-valid",
		"fingerprintHash": "hash-valid"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleFreebuffPoll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if res["status"] != "authorized" {
		t.Errorf("expected status 'authorized', got %v", res["status"])
	}
	if res["connectionId"] != expectedConnID {
		t.Errorf("expected connectionId %s, got %v", expectedConnID, res["connectionId"])
	}

	// Verify database record
	var provider, authType, name, data string
	var isActive int
	err := database.QueryRow(
		"SELECT provider, authType, name, isActive, data FROM providerConnections WHERE id = ?",
		expectedConnID,
	).Scan(&provider, &authType, &name, &isActive, &data)
	if err != nil {
		t.Fatalf("failed to query providerConnections: %v", err)
	}

	if provider != "freebuff" {
		t.Errorf("expected provider 'freebuff', got %s", provider)
	}
	if authType != "oauth" {
		t.Errorf("expected authType 'oauth', got %s", authType)
	}
	if name != "Freebuff Master" {
		t.Errorf("expected name 'Freebuff Master', got %s", name)
	}
	if isActive != 1 {
		t.Errorf("expected isActive 1, got %d", isActive)
	}

	var dataMap map[string]any
	if err := json.Unmarshal([]byte(data), &dataMap); err != nil {
		t.Fatalf("failed to parse connection data: %v", err)
	}
	if dataMap["authToken"] != testAuthToken {
		t.Errorf("expected authToken %s, got %v", testAuthToken, dataMap["authToken"])
	}
	if dataMap["email"] != "user@freebuff.com" {
		t.Errorf("expected email 'user@freebuff.com', got %v", dataMap["email"])
	}
	if dataMap["name"] != "Freebuff Master" {
		t.Errorf("expected name 'Freebuff Master', got %v", dataMap["name"])
	}
}

func TestHandleFreebuffPoll_Authorized_UpdatesExisting(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	repo := db.NewRepo(database)

	testAuthToken := "fb_token_update_test"
	connID := "fb-" + shortHash(testAuthToken)

	// Seed existing connection
	_, err := database.Exec(
		`INSERT INTO providerConnections (id, provider, authType, name, isActive, data, createdAt, updatedAt) VALUES (?, 'freebuff', 'oauth', 'Old Name', 1, '{}', '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z')`,
		connID,
	)
	if err != nil {
		t.Fatalf("failed to seed connection: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "authorized",
			"authToken": "` + testAuthToken + `",
			"name": "Updated Freebuff Name",
			"email": "updated@freebuff.com"
		}`))
	}))
	defer mockServer.Close()

	oldURL := freebuffAuthBaseURL
	freebuffAuthBaseURL = mockServer.URL
	defer func() { freebuffAuthBaseURL = oldURL }()

	handler := NewOAuthHandler(repo)
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/freebuff/poll", strings.NewReader(`{
		"fingerprint_id": "fp-snake",
		"fingerprint_hash": "hash-snake"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleFreebuffPoll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify database record was updated, not duplicated
	var count int
	_ = database.QueryRow("SELECT count(*) FROM providerConnections WHERE id = ?", connID).Scan(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 connection, got %d", count)
	}

	var name, data string
	err = database.QueryRow("SELECT name, data FROM providerConnections WHERE id = ?", connID).Scan(&name, &data)
	if err != nil {
		t.Fatalf("failed to read updated row: %v", err)
	}
	if name != "Updated Freebuff Name" {
		t.Errorf("expected name 'Updated Freebuff Name', got %s", name)
	}
	var dataMap map[string]any
	_ = json.Unmarshal([]byte(data), &dataMap)
	if dataMap["email"] != "updated@freebuff.com" {
		t.Errorf("expected email 'updated@freebuff.com', got %v", dataMap["email"])
	}
}

func TestHandleAntigravityAuthorize(t *testing.T) {
	handler := NewOAuthHandler(nil)

	// JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/antigravity/authorize?state=custom_state_123", nil)
	rec := httptest.NewRecorder()
	handler.HandleAntigravityAuthorize(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	authURL, ok := res["url"].(string)
	if !ok || authURL == "" {
		t.Fatalf("missing or empty url in response")
	}

	if !strings.Contains(authURL, "accounts.google.com") {
		t.Errorf("auth URL should contain accounts.google.com, got %s", authURL)
	}
	if !strings.Contains(authURL, "client_id=") {
		t.Errorf("auth URL should contain client_id, got %s", authURL)
	}
	if !strings.Contains(authURL, "scope=") {
		t.Errorf("auth URL should contain scope, got %s", authURL)
	}
	if !strings.Contains(authURL, "custom_state_123") {
		t.Errorf("auth URL should contain state custom_state_123, got %s", authURL)
	}
	if !strings.Contains(authURL, "access_type=offline") {
		t.Errorf("auth URL should request offline access, got %s", authURL)
	}

	// Redirect response
	reqRedirect := httptest.NewRequest(http.MethodGet, "/api/oauth/antigravity/authorize?redirect=true", nil)
	recRedirect := httptest.NewRecorder()
	handler.HandleAntigravityAuthorize(recRedirect, reqRedirect)

	if recRedirect.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", recRedirect.Code)
	}
	location := recRedirect.Header().Get("Location")
	if !strings.Contains(location, "accounts.google.com") {
		t.Errorf("expected redirect to accounts.google.com, got %s", location)
	}
}

func TestHandleAntigravityCallback_Errors(t *testing.T) {
	handler := NewOAuthHandler(nil)

	// Missing code
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/antigravity/callback", nil)
	rec := httptest.NewRecorder()
	handler.HandleAntigravityCallback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing code, got %d", rec.Code)
	}

	// Error param from Google
	reqErr := httptest.NewRequest(http.MethodGet, "/api/oauth/antigravity/callback?error=access_denied&error_description=user+declined", nil)
	recErr := httptest.NewRecorder()
	handler.HandleAntigravityCallback(recErr, reqErr)
	if recErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oauth error, got %d", recErr.Code)
	}
}

func TestHandleAntigravityCallback_Success(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	repo := db.NewRepo(database)

	mockGoogleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		_ = r.ParseForm()
		code := r.Form.Get("code")
		if code != "valid_google_code_123" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}

		// Simple mock JWT id_token with claims: {"email":"antigravity-user@gmail.com"}
		// Header: {"alg":"none"} -> eyJhbGciOiJub25lIn0
		// Payload: {"email":"antigravity-user@gmail.com"} -> eyJlbWFpbCI6ImFudGlncmF2aXR5LXVzZXJAZ21haWwuY29tIn0
		mockIDToken := "eyJhbGciOiJub25lIn0.eyJlbWFpbCI6ImFudGlncmF2aXR5LXVzZXJAZ21haWwuY29tIn0."

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "ya29.mock_access_token_xyz",
			"refresh_token": "1//mock_refresh_token_uvw",
			"expires_in": 3600,
			"token_type": "Bearer",
			"id_token": "` + mockIDToken + `",
			"scope": "https://www.googleapis.com/auth/cloud-platform"
		}`))
	}))
	defer mockGoogleServer.Close()

	oldTokenURL := googleOAuthTokenURL
	googleOAuthTokenURL = mockGoogleServer.URL + "/token"
	defer func() { googleOAuthTokenURL = oldTokenURL }()

	handler := NewOAuthHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/antigravity/callback?code=valid_google_code_123", nil)
	rec := httptest.NewRecorder()

	handler.HandleAntigravityCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if res["status"] != "authorized" {
		t.Errorf("expected status 'authorized', got %v", res["status"])
	}
	if res["provider"] != "antigravity" {
		t.Errorf("expected provider 'antigravity', got %v", res["provider"])
	}
	if res["email"] != "antigravity-user@gmail.com" {
		t.Errorf("expected email 'antigravity-user@gmail.com', got %v", res["email"])
	}

	connID, ok := res["connectionId"].(string)
	if !ok || connID == "" {
		t.Fatalf("missing connectionId in response: %v", res)
	}

	// Verify DB record
	var provider, authType, name, data string
	var isActive int
	err := database.QueryRow(
		"SELECT provider, authType, name, isActive, data FROM providerConnections WHERE id = ?",
		connID,
	).Scan(&provider, &authType, &name, &isActive, &data)
	if err != nil {
		t.Fatalf("failed to query providerConnections: %v", err)
	}

	if provider != "antigravity" {
		t.Errorf("expected provider 'antigravity', got %s", provider)
	}
	if authType != "oauth" {
		t.Errorf("expected authType 'oauth', got %s", authType)
	}
	if !strings.Contains(name, "antigravity-user@gmail.com") {
		t.Errorf("expected name to contain email, got %s", name)
	}
	if isActive != 1 {
		t.Errorf("expected isActive 1, got %d", isActive)
	}

	var dataMap map[string]any
	if err := json.Unmarshal([]byte(data), &dataMap); err != nil {
		t.Fatalf("failed to unmarshal stored data: %v", err)
	}
	if dataMap["accessToken"] != "ya29.mock_access_token_xyz" {
		t.Errorf("expected accessToken, got %v", dataMap["accessToken"])
	}
	if dataMap["refreshToken"] != "1//mock_refresh_token_uvw" {
		t.Errorf("expected refreshToken, got %v", dataMap["refreshToken"])
	}
	if dataMap["email"] != "antigravity-user@gmail.com" {
		t.Errorf("expected email 'antigravity-user@gmail.com', got %v", dataMap["email"])
	}
}
