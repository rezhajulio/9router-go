package dashboard

import (
	"context"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

func TestHandleGetConnectionModels_Cursor(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	cursorpkg.ClearCursorModelCache()

	// Mock offline fetch
	origFetch := cursorpkg.FetchCursorProtoFunc
	cursorpkg.FetchCursorProtoFunc = func(ctx context.Context, endpointURL string, headers map[string]string) ([]byte, error) {
		m := cursorpkg.ConcatBuffers(
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "cursor-pro-test"),
			cursorpkg.EncodeField(4, cursorpkg.WireBytes, "Cursor Pro Test"),
		)
		payload := cursorpkg.EncodeField(1, cursorpkg.WireBytes, m)
		return payload, nil
	}
	defer func() {
		cursorpkg.FetchCursorProtoFunc = origFetch
		cursorpkg.ClearCursorModelCache()
	}()

	connData := `{"accessToken":"test-tok","providerSpecificData":{"machineId":"m-123"}}`
	if err := repo.CreateProviderConnectionFull("conn-cursor-1", "cursor", "oauth", "Cursor Main", nil, connData); err != nil {
		t.Fatalf("create conn: %v", err)
	}

	router := setupTestRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/providers/conn-cursor-1/models", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Provider string `json:"provider"`
		Models   []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Provider != "cursor" || len(resp.Models) == 0 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	if resp.Models[0].ID != "cursor-pro-test" {
		t.Errorf("expected cursor-pro-test, got %s", resp.Models[0].ID)
	}
}
