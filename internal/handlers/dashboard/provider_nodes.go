package dashboard

import (
	json "encoding/json/v2"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"9router/proxy/internal/handlerutil"
)

// ProviderNodeResponse is the JSON representation of a provider node with unpacked data fields.
type ProviderNodeResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix,omitempty"`
	APIType   string `json:"apiType,omitempty"`
	BaseURL   string `json:"baseUrl,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// HandleGetProviderNodes handles GET /api/provider-nodes.
func (h *DashboardHandler) HandleGetProviderNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.Repo.GetProviderNodes()
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	res := make([]ProviderNodeResponse, 0, len(nodes))
	for _, n := range nodes {
		nodeType := ""
		if n.Type != nil {
			nodeType = *n.Type
		}
		nodeName := ""
		if n.Name != nil {
			nodeName = *n.Name
		}

		item := ProviderNodeResponse{
			ID:        n.ID,
			Type:      nodeType,
			Name:      nodeName,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}

		if n.Data != "" {
			var dataObj struct {
				Prefix  string `json:"prefix"`
				APIType string `json:"apiType"`
				BaseURL string `json:"baseUrl"`
			}
			if err := json.Unmarshal([]byte(n.Data), &dataObj); err == nil {
				item.Prefix = dataObj.Prefix
				item.APIType = dataObj.APIType
				item.BaseURL = dataObj.BaseURL
			}
		}

		res = append(res, item)
	}

	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
		"nodes": res,
	})
}

// HandleCreateProviderNode handles POST /api/provider-nodes.
func (h *DashboardHandler) HandleCreateProviderNode(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Name    string `json:"name"`
		Prefix  string `json:"prefix"`
		APIType string `json:"apiType"`
		BaseURL string `json:"baseUrl"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Prefix = strings.TrimSpace(req.Prefix)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if req.Name == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Prefix == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "prefix is required")
		return
	}

	nodeType := req.Type
	if nodeType == "" {
		nodeType = "openai-compatible"
	}

	apiType := req.APIType
	if apiType == "" {
		apiType = "chat"
	}

	var id string
	if nodeType == "anthropic-compatible" {
		id = "anthropic-compatible-" + uuid.New().String()
		if req.BaseURL == "" {
			req.BaseURL = "https://api.anthropic.com/v1"
		}
	} else {
		id = "openai-compatible-" + apiType + "-" + uuid.New().String()
		if req.BaseURL == "" {
			req.BaseURL = "https://api.openai.com/v1"
		}
	}

	dataBytes, _ := json.Marshal(map[string]string{
		"prefix":  req.Prefix,
		"apiType": apiType,
		"baseUrl": req.BaseURL,
	})

	node, err := h.Repo.CreateProviderNode(id, nodeType, req.Name, string(dataBytes))
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlerutil.WriteJSON(w, http.StatusCreated, map[string]any{
		"node": ProviderNodeResponse{
			ID:        node.ID,
			Type:      nodeType,
			Name:      req.Name,
			Prefix:    req.Prefix,
			APIType:   apiType,
			BaseURL:   req.BaseURL,
			CreatedAt: node.CreatedAt,
			UpdatedAt: node.UpdatedAt,
		},
	})
}

// HandleDeleteProviderNode handles DELETE /api/provider-nodes/{id}.
func (h *DashboardHandler) HandleDeleteProviderNode(w http.ResponseWriter, r *http.Request) {
	id := getURLParam(r, "id")
	if id == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "missing node id")
		return
	}

	if err := h.Repo.DeleteProviderNode(id); err != nil {
		handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
}
