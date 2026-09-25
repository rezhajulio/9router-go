package chat

import (
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"9router/proxy/internal/db"
)

func TestCursorDynamicModelListingFallback(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()

	// Clear seeded connections
	if _, err := database.Exec(`DELETE FROM providerConnections`); err != nil {
		t.Fatalf("delete connections: %v", err)
	}

	repo := db.NewRepo(database)
	handler := NewChatHandler(repo)

	// Create an active Cursor connection with invalid token so live discovery fails open
	connData := `{"accessToken":"bad-token","machineId":"dummy-machine"}`
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
