package chat

import (
	"context"
	json "encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"9router/proxy/internal/db"
	cursorpkg "9router/proxy/internal/proxy/cursor"
)

func TestCursorDynamicModelListing_WithMock(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()

	// Clear seeded connections
	if _, err := database.Exec(`DELETE FROM providerConnections`); err != nil {
		t.Fatalf("delete connections: %v", err)
	}

	cursorpkg.ClearCursorModelCache()

	// Mock offline fetch returning dynamic model
	origFetch := cursorpkg.FetchCursorProtoFunc
	cursorpkg.FetchCursorProtoFunc = func(ctx context.Context, endpointURL string, headers map[string]string) ([]byte, error) {
		m := cursorpkg.ConcatBuffers(
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "cursor-small-free"),
			cursorpkg.EncodeField(4, cursorpkg.WireBytes, "Cursor Small Free"),
		)
		payload := cursorpkg.EncodeField(1, cursorpkg.WireBytes, m)
		return payload, nil
	}
	defer func() {
		cursorpkg.FetchCursorProtoFunc = origFetch
		cursorpkg.ClearCursorModelCache()
	}()

	repo := db.NewRepo(database)
	handler := NewChatHandler(repo)

	// Create an active Cursor connection with nested providerSpecificData.machineId
	connData := `{"accessToken":"valid-token","providerSpecificData":{"machineId":"test-machine-123","ghostMode":true}}`
	if err := repo.CreateProviderConnectionFull("conn-cursor-test", "cursor", "oauth", "Cursor Test", nil, connData); err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()
	handler.HandleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse /v1/models response: %v", err)
	}

	foundDynamic := false
	for _, m := range resp.Data {
		if m.ID == "cu/cursor-small-free" || m.ID == "cursor/cursor-small-free" {
			foundDynamic = true
			break
		}
	}
	if !foundDynamic {
		t.Fatalf("expected dynamic model cursor-small-free to be listed, got: %+v", resp.Data)
	}
}

func TestCursorDynamicModelListingFallback(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()

	// Clear seeded connections
	if _, err := database.Exec(`DELETE FROM providerConnections`); err != nil {
		t.Fatalf("delete connections: %v", err)
	}

	cursorpkg.ClearCursorModelCache()

	// Mock offline fetch returning error
	origFetch := cursorpkg.FetchCursorProtoFunc
	cursorpkg.FetchCursorProtoFunc = func(ctx context.Context, endpointURL string, headers map[string]string) ([]byte, error) {
		return nil, fmt.Errorf("simulated network failure")
	}
	defer func() {
		cursorpkg.FetchCursorProtoFunc = origFetch
		cursorpkg.ClearCursorModelCache()
	}()

	repo := db.NewRepo(database)
	handler := NewChatHandler(repo)

	// Create an active Cursor connection with nested providerSpecificData.machineId
	connData := `{"accessToken":"token","providerSpecificData":{"machineId":"dummy-machine"}}`
	if err := repo.CreateProviderConnectionFull("conn-cursor-test", "cursor", "oauth", "Cursor Test", nil, connData); err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()
	handler.HandleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse /v1/models response: %v", err)
	}

	foundDefault := false
	for _, m := range resp.Data {
		if m.ID == "cu/default" || m.ID == "cursor/default" {
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		t.Fatalf("expected fallback static cursor models to be listed, got: %+v", resp.Data)
	}
}
