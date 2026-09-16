package handlers

import (
	json "encoding/json/v2"
	"net/http"
	"net/http/pprof"
	"github.com/go-chi/chi/v5"

	"9router/proxy/internal/constants"
	"9router/proxy/internal/db"
	"9router/proxy/internal/handlers/chat"
	"9router/proxy/internal/handlers/dashboard"
	"9router/proxy/internal/handlers/media"
	"9router/proxy/internal/handlers/oauth"
	"9router/proxy/internal/handlers/shared"
	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/middleware"
	"9router/proxy/web"
)
// Re-export TokenSaverConfig for root compatibility
type TokenSaverConfig = shared.TokenSaverConfig

// NewTokenSaverConfig re-exports shared.NewTokenSaverConfig.
func NewTokenSaverConfig(rtk, caveman, ponytail bool) *TokenSaverConfig {
	return shared.NewTokenSaverConfig(rtk, caveman, ponytail)
}

// SetupRoutes mounts all domain handlers on the provided router.
func SetupRoutes(r interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
	HandleFunc(pattern string, handlerFn http.HandlerFunc)
}, repo *db.Repo, ts *TokenSaverConfig) {
	chatH := chat.NewChatHandler(repo, ts)
	mediaH := media.NewMediaHandler(repo, ts, chatH)
	oauthH := oauth.NewOAuthHandler(repo)

	dashH := dashboard.NewDashboardHandler(repo)
	// Chat, Version & Models Domain
	r.Get("/version", chatH.HandleVersion)
	r.Get("/api/version", chatH.HandleVersion)
	r.Get("/api/version/status", chatH.HandleVersionStatus)
	r.Get("/api/version/check", chatH.HandleCheckUpdate)
	r.Post("/api/version/update", chatH.HandleTriggerUpdate)
	r.Post("/api/version/auto-update", chatH.HandleToggleAutoUpdate)
	r.Get("/models", chatH.HandleModels)
	r.Get("/models/info", chatH.HandleModelsInfo)
	r.Get("/models/{kind}", chatH.HandleModelsByKind)
	r.Get("/models/*", chatH.HandleModelLookup)
	r.Get("/v1/models", chatH.HandleModels)
	r.Get("/v1/models/*", chatH.HandleModelLookup)
	r.Get("/api/v1/models", chatH.HandleModels)
	r.Get("/api/v1/models/*", chatH.HandleModelLookup)
	r.Get("/api/models/catalog-sync", chatH.HandleCatalogSyncStatus)
	r.Post("/api/models/catalog-sync", chatH.HandleCatalogSyncTrigger)
	r.Post("/chat/completions", chatH.HandleChatCompletions)
	r.Post("/messages", chatH.HandleMessages)
	r.Post("/messages/count_tokens", chatH.HandleCountTokens)
	r.Post("/api/chat", chatH.HandleOllamaChat)

	// Media, Audio, Video & Web Tools Domain
	r.Post("/embeddings", mediaH.HandleEmbeddings)
	r.Post("/responses", mediaH.HandleResponses)
	r.Post("/responses/compact", mediaH.HandleResponsesCompact)
	r.Post("/images/generations", mediaH.HandleImages)
	r.Post("/audio/speech", mediaH.HandleAudioSpeech)
	r.Get("/audio/voices", mediaH.HandleAudioVoices)
	r.Post("/audio/transcriptions", mediaH.HandleAudioTranscriptions)
	r.Post("/videos/generations", mediaH.HandleVideoGenerations)
	r.Post("/videos/edits", mediaH.HandleVideoEdits)
	r.Post("/videos/extensions", mediaH.HandleVideoExtensions)
	r.Get("/videos/{id}", mediaH.HandleVideoGet)
	r.Post("/search", mediaH.HandleSearch)
	r.Post("/scrape", mediaH.HandleScrape)
	r.Post("/web/fetch", mediaH.HandleWebFetch)

	// Proxy Pool Deploy Domain
	r.Post("/proxy-pools/vercel-deploy", mediaH.HandleVercelDeploy)
	r.Post("/proxy-pools/deno-deploy", mediaH.HandleDenoDeploy)
	r.Post("/proxy-pools/cloudflare-deploy", mediaH.HandleCloudflareDeploy)

	// CLI Tools Status Domain (dashboard batch status for installed CLI tools)
	r.Get("/cli-tools/all-statuses", media.NewCLIToolsHandler().HandleAllStatuses)

	// Headroom Management Domain (token-compression proxy lifecycle + dashboard proxy)
	headroomH := media.NewHeadroomHandler(repo)
	r.Post("/headroom/start", headroomH.HandleHeadroomStart)
	r.Post("/headroom/stop", headroomH.HandleHeadroomStop)
	r.Post("/headroom/restart", headroomH.HandleHeadroomRestart)
	r.Get("/headroom/status", headroomH.HandleHeadroomStatus)
	r.Get("/headroom/extras", headroomH.HandleHeadroomExtras)
	r.Post("/headroom/extras", headroomH.HandleHeadroomExtras)
	r.Delete("/headroom/extras", headroomH.HandleHeadroomExtras)
	r.HandleFunc("/headroom/proxy", headroomH.HandleHeadroomProxy)
	r.HandleFunc("/headroom/proxy/*", headroomH.HandleHeadroomProxy)

	// OAuth & Import Tokens Domain
	r.Post("/api/oauth/{provider}/import", oauthH.HandleOAuthImport)
	r.Get("/api/oauth/kiro/social-authorize", oauthH.HandleOAuthKiroSocialAuthorize)
	r.Post("/api/oauth/kiro/social-exchange", oauthH.HandleOAuthKiroSocialExchange)
	r.Post("/api/oauth/codex/bulk-import", oauthH.HandleOAuthCodexBulkImport)
	r.Post("/api/oauth/grok-cli/bulk-import", oauthH.HandleOAuthGrokCliBulkImport)
	r.Post("/api/oauth/freebuff/initiate", oauthH.HandleFreebuffInitiate)
	r.Post("/api/oauth/freebuff/poll", oauthH.HandleFreebuffPoll)
	r.Get("/api/oauth/antigravity/authorize", oauthH.HandleAntigravityAuthorize)
	r.Get("/api/oauth/antigravity/callback", oauthH.HandleAntigravityCallback)
	r.Post("/api/oauth/antigravity/callback", oauthH.HandleAntigravityCallback)

	// Live Console Logs Domain (dashboard "Monitor Console Log")
	r.Get("/translator/console-logs", HandleConsoleLogsGet)
	r.Delete("/translator/console-logs", HandleConsoleLogsDelete)
	r.Get("/translator/console-logs/stream", HandleConsoleLogsStream)

	// Usage Real-time SSE Stream & Stats Domain (dashboard topology animation + recent requests)
	r.Get("/usage/stream", HandleUsageStream(repo))
	r.Get("/api/usage/stream", HandleUsageStream(repo))
	r.Get("/usage/stats", HandleUsageStats(repo))
	r.Get("/api/usage/stats", HandleUsageStats(repo))
	r.Get("/api/usage/request-details", HandleRequestDetails(repo))

	// Debug Tracing Domain (p50/p95 latency per provider+model)
	r.Get("/debug/traces", HandleDebugTraces)

	// Dashboard REST API Domain
	r.Get("/api/connections", dashH.HandleGetConnections)
	r.Post("/api/connections", dashH.HandleCreateConnection)
	r.Put("/api/connections/{id}", dashH.HandleUpdateConnection)
	r.Delete("/api/connections/{id}", dashH.HandleDeleteConnection)

	r.Get("/api/provider-nodes", dashH.HandleGetProviderNodes)
	r.Post("/api/provider-nodes", dashH.HandleCreateProviderNode)
	r.Delete("/api/provider-nodes/{id}", dashH.HandleDeleteProviderNode)

	r.Get("/api/combos", dashH.HandleGetCombos)
	r.Post("/api/combos", dashH.HandleCreateCombo)
	r.Put("/api/combos/{id}", dashH.HandleUpdateCombo)
	r.Delete("/api/combos/{id}", dashH.HandleDeleteCombo)

	r.Get("/api/keys", dashH.HandleGetApiKeys)
	r.Post("/api/keys", dashH.HandleCreateApiKey)
	r.Delete("/api/keys/{id}", dashH.HandleDeleteApiKey)
	r.Put("/api/keys/{id}/toggle", dashH.HandleToggleApiKey)

	r.Get("/api/models/custom", dashH.HandleGetCustomModels)
	r.Post("/api/models/custom", dashH.HandleSaveCustomModel)
	r.Delete("/api/models/custom/{key}", dashH.HandleDeleteCustomModel)
	r.Get("/api/models/disabled", dashH.HandleGetDisabledModels)
	r.Put("/api/models/disabled/{provider}", dashH.HandleSaveDisabledModels)

	r.Get("/api/settings", dashH.HandleGetSettings)
	r.Put("/api/settings", dashH.HandleUpdateSettings)
}

// SetupServerRouter mounts public endpoints (/health, /api/hello) and
// API-key protected routes (all engine + admin routes) on the chi router.
func SetupServerRouter(r chi.Router, repo *db.Repo, ts *TokenSaverConfig) {
	// Public (unauthenticated) endpoints
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Embedded Native Dashboard SPA
	webH := web.Handler()
	r.Get("/", webH.ServeHTTP)
	r.Get("/dashboard", webH.ServeHTTP)
	r.Get("/dashboard/*", webH.ServeHTTP)
	r.Get("/connections", webH.ServeHTTP)
	r.Get("/combos", webH.ServeHTTP)
	r.Get("/analytics", webH.ServeHTTP)
	r.Get("/settings", webH.ServeHTTP)
	r.Get("/keys", webH.ServeHTTP)
	r.HandleFunc("/assets/*", webH.ServeHTTP)
	r.HandleFunc("/providers/*", webH.ServeHTTP)
	r.Get("/favicon.ico", webH.ServeHTTP)

	r.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			w.Write([]byte(`{"status":"ok","message":"hello"}`))
		}
	})

	// Profiling endpoints (pprof)
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.HandleFunc("/debug/pprof/*", pprof.Index)

	// API-key protected domain routes (includes /admin/health/reset so health
	// state cannot be reset by an unauthenticated caller — open-source hardening)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireApiKey(repo))

		// Health reset endpoint — dashboard calls this via headroom proxy
		r.Post("/admin/health/reset", func(w http.ResponseWriter, r *http.Request) {
			provider := r.URL.Query().Get("provider")
			model := r.URL.Query().Get("model")
			if err := repo.ResetProviderHealth(provider, model); err != nil {
				handlerutil.WriteJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
			json.MarshalWrite(w, map[string]string{"status": "ok"})
		})

		SetupRoutes(r, repo, ts)
	})
}
