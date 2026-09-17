package oauth

import (
	json "encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/log"
	"9router/proxy/internal/models"
)

var freebuffAPIBaseURL = "https://www.codebuff.com"

// HandleFreebuffSessionStatus returns current active Freebuff session status for a connection.
// GET /api/oauth/freebuff/session
func (h *OAuthHandler) HandleFreebuffSessionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handlerutil.WriteJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	connectionID := r.URL.Query().Get("connectionId")
	var conn *models.ProviderConnection

	if connectionID != "" {
		if h.Repo != nil {
			var err error
			conn, err = h.Repo.GetProviderConnectionByID(connectionID)
			if err != nil {
				log.Error("oauth", "failed to query connection by id", "conn", connectionID, "error", err)
				handlerutil.WriteJSONError(w, http.StatusInternalServerError, "failed to query connection")
				return
			}
		}
	} else {
		if h.Repo != nil {
			conns, err := h.Repo.GetProviderConnections("freebuff", true)
			if err != nil {
				log.Error("oauth", "failed to query freebuff connections", "error", err)
				handlerutil.WriteJSONError(w, http.StatusInternalServerError, "failed to query connections")
				return
			}
			if len(conns) > 0 {
				conn = conns[0]
			}
		}
	}

	if conn == nil {
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "none",
		})
		return
	}

	var data struct {
		AuthToken   string `json:"authToken"`
		AccessToken string `json:"accessToken"`
		APIKey      string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(conn.Data), &data); err != nil {
		log.Error("oauth", "failed to unmarshal connection data", "conn", conn.ID, "error", err)
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "none",
		})
		return
	}

	authToken := data.AuthToken
	if authToken == "" {
		authToken = data.AccessToken
	}
	if authToken == "" {
		authToken = data.APIKey
	}

	if authToken == "" {
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "none",
		})
		return
	}

	reqURL := freebuffAPIBaseURL + "/api/v1/freebuff/session"
	upReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("create session request failed: %v", err))
		return
	}
	upReq.Header.Set("Authorization", "Bearer "+authToken)
	upReq.Header.Set("User-Agent", "codebuff-cli/0.0.138")
	upReq.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(upReq)
	if err != nil {
		log.Error("oauth", "freebuff session request failed", "error", err)
		handlerutil.WriteJSONError(w, http.StatusBadGateway, fmt.Sprintf("freebuff session request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "unauthorized",
		})
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "none",
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadGateway, "failed to read freebuff session response")
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn("oauth", "freebuff session non-200 response", "status", resp.StatusCode, "body", string(body))
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "none",
		})
		return
	}

	var upstream struct {
		Status       string         `json:"status"`
		CurrentModel string         `json:"currentModel"`
		Model        string         `json:"model"`
		InstanceID   string         `json:"instanceId"`
		ExpiresAt    string         `json:"expiresAt"`
		Freebucks    map[string]any `json:"freebucks"`
	}
	if err := json.Unmarshal(body, &upstream); err != nil {
		log.Error("oauth", "freebuff session unmarshal failed", "error", err)
		handlerutil.WriteJSONError(w, http.StatusBadGateway, "failed to parse freebuff session response")
		return
	}

	currentModel := upstream.CurrentModel
	if currentModel == "" {
		currentModel = upstream.Model
	}

	status := strings.ToLower(strings.TrimSpace(upstream.Status))
	if status == "" {
		if currentModel != "" || upstream.InstanceID != "" {
			status = "active"
		} else {
			status = "none"
		}
	} else if status != "active" && status != "unauthorized" {
		status = "none"
	}

	respMap := map[string]any{
		"status": status,
	}
	if currentModel != "" {
		respMap["currentModel"] = currentModel
	}
	if upstream.InstanceID != "" {
		respMap["instanceId"] = upstream.InstanceID
	}
	if upstream.ExpiresAt != "" {
		respMap["expiresAt"] = upstream.ExpiresAt
	}
	if upstream.Freebucks != nil {
		respMap["freebucks"] = upstream.Freebucks
	}

	handlerutil.WriteJSON(w, http.StatusOK, respMap)
}
