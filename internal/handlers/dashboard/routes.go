package dashboard

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"9router/proxy/internal/db"
)

// DashboardHandler handles dashboard REST API endpoints.
type DashboardHandler struct {
	Repo *db.Repo
}

// NewDashboardHandler initializes a DashboardHandler with the provided Repo.
func NewDashboardHandler(repo *db.Repo) *DashboardHandler {
	return &DashboardHandler{Repo: repo}
}

// getURLParam retrieves a route parameter from Chi URLParam or standard PathValue.
func getURLParam(r *http.Request, key string) string {
	if v := chi.URLParam(r, key); v != "" {
		return v
	}
	return r.PathValue(key)
}

// RegisterRoutes registers all dashboard REST endpoints under /api on the router.
func RegisterRoutes(r chi.Router, h *DashboardHandler) {
	r.Route("/api", func(r chi.Router) {
		// Connections
		r.Get("/connections", h.HandleGetConnections)
		r.Post("/connections", h.HandleCreateConnection)
		r.Put("/connections/{id}", h.HandleUpdateConnection)
		r.Put("/providers/{id}", h.HandleUpdateConnection)
		r.Delete("/connections/{id}", h.HandleDeleteConnection)

		// Provider Nodes (Custom Endpoints)
		r.Get("/provider-nodes", h.HandleGetProviderNodes)
		r.Post("/provider-nodes", h.HandleCreateProviderNode)
		r.Delete("/provider-nodes/{id}", h.HandleDeleteProviderNode)

		// Combos
		r.Get("/combos", h.HandleGetCombos)
		r.Post("/combos", h.HandleCreateCombo)
		r.Put("/combos/{id}", h.HandleUpdateCombo)
		r.Delete("/combos/{id}", h.HandleDeleteCombo)

		// API Keys
		r.Get("/keys", h.HandleGetApiKeys)
		r.Post("/keys", h.HandleCreateApiKey)
		r.Delete("/keys/{id}", h.HandleDeleteApiKey)
		r.Put("/keys/{id}/toggle", h.HandleToggleApiKey)

		// Models
		r.Get("/models/custom", h.HandleGetCustomModels)
		r.Post("/models/custom", h.HandleSaveCustomModel)
		r.Delete("/models/custom/{key}", h.HandleDeleteCustomModel)
		r.Get("/models/disabled", h.HandleGetDisabledModels)
		r.Put("/models/disabled/{provider}", h.HandleSaveDisabledModels)

		// Settings
		r.Get("/settings", h.HandleGetSettings)
		r.Put("/settings", h.HandleUpdateSettings)
	})
}
