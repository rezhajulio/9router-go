package dashboard

import (
	json "encoding/json/v2"
	"io"
	"net/http"

	"github.com/google/uuid"

	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/models"
)

// HandleGetConnections handles GET /api/connections.
// Returns a JSON list of all connections.
func (h *DashboardHandler) HandleGetConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := h.Repo.GetProviderConnections("", false)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conns == nil {
		conns = []*models.ProviderConnection{}
	}
	handlerutil.WriteJSON(w, http.StatusOK, conns)
}

// HandleCreateConnection handles POST /api/connections.
// Parses id, provider, authType, name, apiKey, data and calls CreateProviderConnection.
func (h *DashboardHandler) HandleCreateConnection(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		ID       string `json:"id"`
		Provider string `json:"provider"`
		AuthType string `json:"authType"`
		Name     string `json:"name"`
		APIKey   string `json:"apiKey"`
		Data     any    `json:"data"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
	}

	if req.Provider == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "missing provider")
		return
	}
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.AuthType == "" {
		req.AuthType = "apikey"
	}

	apiKey := req.APIKey
	var dataStr string
	if req.Data != nil {
		switch d := req.Data.(type) {
		case string:
			dataStr = d
			if apiKey == "" {
				var m map[string]any
				if err := json.Unmarshal([]byte(d), &m); err == nil {
					if k, ok := m["apiKey"].(string); ok {
						apiKey = k
					}
				}
			}
		case map[string]any:
			if apiKey == "" {
				if k, ok := d["apiKey"].(string); ok {
					apiKey = k
				}
			}
			b, err := json.Marshal(d)
			if err == nil {
				dataStr = string(b)
			}
		default:
			b, err := json.Marshal(d)
			if err == nil {
				dataStr = string(b)
			}
		}
	}

	if err := h.Repo.CreateProviderConnection(req.ID, req.Provider, req.AuthType, req.Name, apiKey); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if dataStr != "" {
		_ = h.Repo.UpdateProviderConnection(req.ID, req.Name, 0, true, dataStr)
	}

	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"id":     req.ID,
	})
}

// HandleUpdateConnection handles PUT /api/connections/{id}.
// Parses name, priority, isActive, data and calls UpdateProviderConnection or SetConnectionStatus.
func (h *DashboardHandler) HandleUpdateConnection(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	if id == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "missing connection id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var rawBody map[string]any
	if err := json.Unmarshal(body, &rawBody); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	existing, err := h.Repo.GetProviderConnectionByID(id)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		handlerutil.WriteJSONError(w, http.StatusNotFound, "connection not found")
		return
	}

	_, hasName := rawBody["name"]
	_, hasPriority := rawBody["priority"]
	_, hasData := rawBody["data"]
	_, hasPSD := rawBody["providerSpecificData"]
	_, hasAssignedModel := rawBody["assignedModel"]
	a, hasIsActive := rawBody["isActive"].(bool)

	// Fast path: if only updating active status
	if hasIsActive && !hasName && !hasPriority && !hasData && !hasPSD && !hasAssignedModel {
		if err := h.Repo.SetConnectionStatus(id, a); err != nil {
			handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		handlerutil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id, "isActive": a})
		return
	}

	name := ""
	if n, ok := rawBody["name"].(string); ok && n != "" {
		name = n
	} else if existing.Name != nil {
		name = *existing.Name
	}
	priority := 0
	if p, ok := rawBody["priority"].(float64); ok {
		priority = int(p)
	} else if existing.Priority != nil {
		priority = *existing.Priority
	}
	isActive := existing.IsActive == 1
	if hasIsActive {
		isActive = a
	}

	dataStr := existing.Data
	if hasData || hasPSD || hasAssignedModel {
		dataMap := make(map[string]any)
		if existing.Data != "" {
			_ = json.Unmarshal([]byte(existing.Data), &dataMap)
		}

		if hasData && rawBody["data"] != nil {
			switch d := rawBody["data"].(type) {
			case string:
				var m map[string]any
				if err := json.Unmarshal([]byte(d), &m); err == nil {
					for k, v := range m {
						if k == "providerSpecificData" {
							if psd, ok := v.(map[string]any); ok {
								mergeMapField(dataMap, "providerSpecificData", psd)
								continue
							}
						}
						dataMap[k] = v
					}
				} else {
					dataStr = d
				}
			case map[string]any:
				for k, v := range d {
					if k == "providerSpecificData" {
						if psd, ok := v.(map[string]any); ok {
							mergeMapField(dataMap, "providerSpecificData", psd)
							continue
						}
					}
					dataMap[k] = v
				}
			}
		}

		if hasPSD {
			if psd, ok := rawBody["providerSpecificData"].(map[string]any); ok {
				mergeMapField(dataMap, "providerSpecificData", psd)
			}
		}

		if hasAssignedModel {
			if am, ok := rawBody["assignedModel"].(string); ok {
				dataMap["assignedModel"] = am
			} else if rawBody["assignedModel"] == nil {
				delete(dataMap, "assignedModel")
			}
		}

		if b, err := json.Marshal(dataMap); err == nil {
			dataStr = string(b)
		}
	}

	if err := h.Repo.UpdateProviderConnection(id, name, priority, isActive, dataStr); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
}

func mergeMapField(target map[string]any, key string, source map[string]any) {
	existing, _ := target[key].(map[string]any)
	if existing == nil {
		existing = make(map[string]any)
	}
	for k, v := range source {
		existing[k] = v
	}
	target[key] = existing
}

// HandleDeleteConnection handles DELETE /api/connections/{id}.
// Calls DeleteProviderConnection.
func (h *DashboardHandler) HandleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	if id == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "missing connection id")
		return
	}

	if err := h.Repo.DeleteProviderConnection(id); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
}
