package dashboard

import (
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTunnelEndpoints(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	h := NewDashboardHandler(repo)

	t.Run("TunnelStatus_Empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
		rec := httptest.NewRecorder()

		h.HandleTunnelStatus(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res struct {
			Tunnel struct {
				Enabled   bool   `json:"enabled"`
				TunnelURL string `json:"tunnelUrl"`
				Running   bool   `json:"running"`
			} `json:"tunnel"`
			Tailscale struct {
				Enabled bool `json:"enabled"`
			} `json:"tailscale"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal status response: %v", err)
		}

		if res.Tunnel.Enabled || res.Tunnel.Running {
			t.Errorf("expected tunnel to be disabled and not running initially")
		}
	})

	t.Run("TunnelStatus_WithSettings", func(t *testing.T) {
		_ = repo.UpdateSettingsRaw(map[string]any{
			"tunnelEnabled": true,
			"tunnelUrl":     "https://test.trycloudflare.com",
		})

		req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
		rec := httptest.NewRecorder()

		h.HandleTunnelStatus(rec, req)

		var res struct {
			Tunnel struct {
				Enabled   bool   `json:"enabled"`
				TunnelURL string `json:"tunnelUrl"`
				Running   bool   `json:"running"`
			} `json:"tunnel"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal status response: %v", err)
		}

		if !res.Tunnel.Enabled || !res.Tunnel.Running {
			t.Errorf("expected tunnel to be enabled and running when url is configured")
		}
		if res.Tunnel.TunnelURL != "https://test.trycloudflare.com" {
			t.Errorf("unexpected tunnel URL: %s", res.Tunnel.TunnelURL)
		}
	})

	t.Run("TunnelEnable_ReturnsCleanError", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/tunnel/enable", nil)
		rec := httptest.NewRecorder()

		h.HandleTunnelEnable(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var res struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error response: %v", err)
		}
		if res.Error == "" {
			t.Errorf("expected non-empty error message")
		}
	})

	t.Run("TunnelDisable_Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/tunnel/disable", nil)
		rec := httptest.NewRecorder()

		h.HandleTunnelDisable(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal disable response: %v", err)
		}
		if !res.Success {
			t.Errorf("expected success=true")
		}
	})

	t.Run("TailscaleCheck", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tunnel/tailscale-check", nil)
		rec := httptest.NewRecorder()

		h.HandleTailscaleCheck(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res struct {
			Installed bool `json:"installed"`
			LoggedIn  bool `json:"loggedIn"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal check response: %v", err)
		}
	})

	t.Run("TailscaleEnable_ReturnsCleanError", func(t *testing.T) {
		orig := unixTailscaleCandidates
		unixTailscaleCandidates = nil
		defer func() { unixTailscaleCandidates = orig }()
		t.Setenv("PATH", "")

		req := httptest.NewRequest(http.MethodPost, "/api/tunnel/tailscale-enable", nil)
		rec := httptest.NewRecorder()

		h.HandleTailscaleEnable(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var res struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error response: %v", err)
		}
		if res.Error == "" {
			t.Errorf("expected non-empty error message")
		}
	})

	t.Run("TailscaleDisable_Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/tunnel/tailscale-disable", nil)
		rec := httptest.NewRecorder()

		h.HandleTailscaleDisable(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal disable response: %v", err)
		}
		if !res.Success {
			t.Errorf("expected success=true")
		}
	})
}
