package chat

import (
	json "encoding/json/v2"
	"fmt"
	"strings"
	"9router/proxy/internal/constants"
	"9router/proxy/internal/db"
	"9router/proxy/internal/log"
	"9router/proxy/internal/models"
	"9router/proxy/internal/providers"
	internalproxy "9router/proxy/internal/proxy"
)

// CredentialFallbacks maps search/tool providers to the primary chat provider whose API key can be reused.
var CredentialFallbacks = map[string]string{
	"ollama-search": "ollama",
	"zai-search":    "glm",
	"cline":         "clinepass",
	"clinepass":     "cline",
}


// GetBestConnection retrieves the highest-priority active connection for a provider.
// When connectionID is non-empty, it fetches that specific connection directly.
func (h *ChatHandler) GetBestConnection(provider string, connectionID string, excludeIDs []string, model string) (*models.ProviderConnection, *ConnectionData, error) {
	return h.getBestConnection(provider, connectionID, excludeIDs, model)
}

func (h *ChatHandler) getBestConnection(provider string, connectionID string, excludeIDs []string, model string) (*models.ProviderConnection, *ConnectionData, error) {
	if model != "" && !h.Repo.IsProviderAvailable(provider, model) {
		log.Warn("health", "unhealthy provider", "provider", provider, "model", model)
	}

	var conn *models.ProviderConnection
	var err error

	if connectionID != "" {
		conn, err = h.Repo.GetProviderConnectionByID(connectionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch connection %s: %w", connectionID, err)
		}
		if conn == nil {
			return nil, nil, fmt.Errorf("connection %s not found", connectionID)
		}
	} else {
		connections, queryErr := h.Repo.GetProviderConnections(provider, true)
		if queryErr != nil {
			return nil, nil, fmt.Errorf("failed to query connections for %s: %w", provider, queryErr)
		}
		if len(connections) == 0 {
			if fallbackProvider, ok := CredentialFallbacks[provider]; ok {
				fallbackConns, fallbackErr := h.Repo.GetProviderConnections(fallbackProvider, true)
				if fallbackErr == nil && len(fallbackConns) > 0 {
					connections = fallbackConns
				}
			}
		}
		if len(connections) == 0 {
			if cfg, ok := providers.KnownProviders[provider]; ok && cfg.NoAuth {
				// Inject virtual connection for no-auth provider with optional proxy pool strategy from settings
				connData := &ConnectionData{
					AccessToken: "public",
				}
				settings, err := h.Repo.GetSettings()
				if err == nil && settings != nil && settings.ProviderStrategies != nil {
					if strat, ok := settings.ProviderStrategies[provider]; ok {
						if strat.ProxyPoolID != "" && strat.ProxyPoolID != "__none__" {
							connData.ProxyPoolID = strat.ProxyPoolID
						}
					}
				}
				publicName := "Public"
				conn := &models.ProviderConnection{
					ID:       "noauth",
					Provider: provider,
					Name:     &publicName,
					IsActive: 1,
				}
				return conn, connData, nil
			}
			return nil, nil, fmt.Errorf("no active connections for provider: %s", provider)
		}

		settings, _ := h.Repo.GetSettings()
		connections = filterConnectionsForModel(provider, connections, model, settings)
		if len(connections) == 0 {
			return nil, nil, fmt.Errorf("no connection assigned to model %s for provider %s under strict assignment", model, provider)
		}

		excludeSet := make(map[string]bool, len(excludeIDs))
		for _, id := range excludeIDs {
			excludeSet[id] = true
		}

		conn = nil
		for _, c := range connections {
			if excludeSet[c.ID] {
				continue
			}
			// Skip connections that have an active per-connection model lock
			if model != "" {
				if locked, _ := h.Repo.IsConnectionModelLocked(c.ID, model); locked {
					continue
				}
				if provider == "antigravity" && IsAntigravityModelBlocked(c.ID, model) {
					continue
				}
			}
			conn = c
			break
		}
		if conn == nil {
			return nil, nil, fmt.Errorf("no available connections for provider: %s (all excluded)", provider)
		}
	}

	var connData ConnectionData
	if conn.Data != "" {
		if err := json.Unmarshal([]byte(conn.Data), &connData); err != nil {
			return nil, nil, fmt.Errorf("failed to parse connection data: %w", err)
		}
	}

	return conn, &connData, nil
}

// GetProviderConfig returns the upstream configuration for a provider.
func (h *ChatHandler) GetProviderConfig(provider string, connData *ConnectionData) (*providers.ProviderConfig, error) {
	return h.getProviderConfig(provider, connData)
}

func (h *ChatHandler) getProviderConfig(provider string, connData *ConnectionData) (*providers.ProviderConfig, error) {
	var baseCfg *providers.ProviderConfig

	if connData != nil && connData.BaseURL != "" {
		if cfg, ok := providers.KnownProviders[provider]; ok {
			cloned := cfg
			cloned.BaseURL = connData.BaseURL
			baseCfg = &cloned
		} else {
			baseCfg = &providers.ProviderConfig{
				BaseURL:    connData.BaseURL,
				AuthHeader: constants.HeaderAuthorization,
				AuthScheme: constants.AuthSchemeBearer,
			}
		}
	} else if cfg, ok := providers.KnownProviders[provider]; ok {
		// Clone config so per-request headers don't mutate global registry
		cloned := cfg
		baseCfg = &cloned
	} else {
		node, nodeData, err := h.Repo.GetProviderNodeByID(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to look up provider node %s: %w", provider, err)
		}
		if node != nil && nodeData != nil && nodeData.BaseURL != "" {
			baseURL := nodeData.BaseURL
			if !strings.HasSuffix(baseURL, "/chat/completions") {
				if strings.HasSuffix(baseURL, "/v1") || strings.HasSuffix(baseURL, "/v1/") {
					baseURL = strings.TrimRight(baseURL, "/") + "/chat/completions"
				} else {
					baseURL = strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
				}
			}
			baseCfg = &providers.ProviderConfig{
				BaseURL:    baseURL,
				AuthHeader: constants.HeaderAuthorization,
				AuthScheme: constants.AuthSchemeBearer,
			}
		}
	}

	if baseCfg == nil {
		return nil, fmt.Errorf("provider %q has no baseUrl in connection data and is not in KnownProviders", provider)
	}

	// Check if this connection uses an Edge Relay Proxy Pool (Vercel, Cloudflare, Deno)
	if connData != nil {
		var relayURL string
		var noProxy string

		if connData.ProxyPoolID != "" {
			if pool, err := h.Repo.GetProxyPool(connData.ProxyPoolID); err == nil && pool != nil && pool.IsActive {
				if pool.Type == "vercel" || pool.Type == "cloudflare" || pool.Type == "deno" {
					relayURL = pool.NextURL()
					noProxy = pool.NoProxy
				}
			}
		}
		if relayURL == "" && connData.ProviderSpecificData != nil {
			if u, ok := connData.ProviderSpecificData["vercelRelayUrl"].(string); ok && u != "" {
				relayURL = u
				if np, ok := connData.ProviderSpecificData["connectionNoProxy"].(string); ok {
					noProxy = np
				}
			}
		}

		if relayURL != "" && !internalproxy.ShouldBypassNoProxy(baseCfg.BaseURL, noProxy) {
			cloned := *baseCfg
			cloned.StaticHeaders = internalproxy.BuildEdgeRelayHeaders(baseCfg.BaseURL, cloned.StaticHeaders)
			cloned.BaseURL = relayURL
			return &cloned, nil
		}
	}

	return baseCfg, nil
}

// ExtractAPIKey gets the API key from a connection's data.
func ExtractAPIKey(connData *ConnectionData) string {
	return extractAPIKey(connData)
}

func extractAPIKey(connData *ConnectionData) string {
	if connData.APIKey != "" {
		return connData.APIKey
	}
	return connData.AccessToken
}

// NormalizeProviderToken normalizes credentials for providers with specific token requirements
// (e.g. Cline OAuth tokens require workos: prefix, whereas API keys ride plain Bearer).
func NormalizeProviderToken(provider, token string) string {
	if (provider == "cline" || provider == "clinepass") && token != "" {
		t := strings.TrimSpace(token)
		if !strings.HasPrefix(t, "workos:") && !strings.HasPrefix(t, "sk_") {
			return "workos:" + t
		}
		return t
	}
	return token
}


func extractAssignedModel(dataStr string) string {
	if dataStr == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(dataStr), &m); err != nil {
		return ""
	}
	if am, ok := m["assignedModel"].(string); ok && am != "" {
		return am
	}
	if fm, ok := m["freebuffModel"].(string); ok && fm != "" {
		return fm
	}
	if psd, ok := m["providerSpecificData"].(map[string]any); ok {
		if am, ok := psd["assignedModel"].(string); ok && am != "" {
			return am
		}
		if fm, ok := psd["freebuffModel"].(string); ok && fm != "" {
			return fm
		}
	}
	return ""
}

func filterConnectionsForModel(provider string, connections []*models.ProviderConnection, model string, settings *db.SettingsData) []*models.ProviderConnection {
	if settings == nil || settings.ProviderStrategies == nil || model == "" {
		return connections
	}
	strat, ok := settings.ProviderStrategies[provider]
	if !ok || !strat.StrictModelAssignment {
		return connections
	}

	var filtered []*models.ProviderConnection
	for _, c := range connections {
		if c == nil {
			continue
		}
		if extractAssignedModel(c.Data) == model {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
