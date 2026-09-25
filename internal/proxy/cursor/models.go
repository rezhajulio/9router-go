package cursor

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

const (
	cursorModelsEndpoint = "/agent.v1.AgentService/GetUsableModels"
	cursorModelsHost     = "agent.api5.cursor.sh"
	cursorModelsURL      = "https://agent.api5.cursor.sh" + cursorModelsEndpoint
	modelsCacheTTL       = 5 * time.Minute
)

// ModelEntry represents a single discovered Cursor model.
type ModelEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cachedCatalog struct {
	expiresAt time.Time
	models    []ModelEntry
}

var (
	catalogCacheMu sync.RWMutex
	catalogCache   = make(map[string]cachedCatalog)
)

// ClearCursorModelCache clears the memory cache.
func ClearCursorModelCache() {
	catalogCacheMu.Lock()
	defer catalogCacheMu.Unlock()
	catalogCache = make(map[string]cachedCatalog)
}

func catalogCacheKey(accessToken, machineID string) string {
	seed := machineID + ":" + accessToken
	h := sha256.Sum256([]byte("cursor:" + seed))
	return hex.EncodeToString(h[:])
}

// ParseCursorUsableModels decodes agent.v1.GetUsableModelsResponse protobuf payload.
// Response contains repeated ModelDetails (field 1):
// - field 1: model id
// - field 3: display model id
// - field 4: display name
// - field 5: display name short
func ParseCursorUsableModels(payload []byte) []ModelEntry {
	msg := DecodeMessage(payload)
	seen := make(map[string]bool)
	var models []ModelEntry

	for _, entry := range msg.Get(1) { // 1 = repeated ModelDetails
		if entry.WireType != WireBytes || len(entry.Value) == 0 {
			continue
		}
		detail := DecodeMessage(entry.Value)
		id := ""
		if detail.Has(1) {
			id = strings.TrimSpace(string(detail.Get(1)[0].Value))
		}
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true

		name := ""
		if detail.Has(4) {
			name = strings.TrimSpace(string(detail.Get(4)[0].Value))
		} else if detail.Has(5) {
			name = strings.TrimSpace(string(detail.Get(5)[0].Value))
		} else if detail.Has(3) {
			name = strings.TrimSpace(string(detail.Get(3)[0].Value))
		} else {
			name = id
		}

		models = append(models, ModelEntry{
			ID:   id,
			Name: name,
		})
	}

	return models
}

// ResolveCursorModels fetches live usable models for the given credentials.
// Unary GetUsableModels on agent.api5.cursor.sh is HTTP/2 only with content-type: application/proto.
func ResolveCursorModels(ctx context.Context, accessToken, machineID string, ghostMode bool, forceRefresh bool) ([]ModelEntry, error) {
	if accessToken == "" || machineID == "" {
		return nil, nil
	}

	key := catalogCacheKey(accessToken, machineID)
	now := time.Now()

	if !forceRefresh {
		catalogCacheMu.RLock()
		cached, ok := catalogCache[key]
		catalogCacheMu.RUnlock()
		if ok && cached.expiresAt.After(now) {
			return cached.models, nil
		}
	}

	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	headers := BuildCursorHeaders(accessToken, machineID, ghostMode)
	headers["accept"] = "application/proto"
	headers["content-type"] = "application/proto"
	delete(headers, "connect-accept-encoding")
	delete(headers, "connect-protocol-version")

	payload, err := fetchCursorProtoH2(reqCtx, cursorModelsURL, headers)
	if err != nil {
		return nil, err
	}

	models := ParseCursorUsableModels(payload)
	if len(models) == 0 {
		return nil, nil
	}

	catalogCacheMu.Lock()
	catalogCache[key] = cachedCatalog{
		expiresAt: now.Add(modelsCacheTTL),
		models:    models,
	}
	catalogCacheMu.Unlock()

	return models, nil
}

func fetchCursorProtoH2(ctx context.Context, endpointURL string, headers map[string]string) ([]byte, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, err
	}

	host := u.Host
	port := "443"
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err == nil {
			host = h
			port = p
		}
	}

	tlsConfig := &tls.Config{
		ServerName: host,
		NextProtos: []string{"h2"},
	}

	d := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, err := tls.DialWithDialer(d, "tcp", net.JoinHostPort(host, port), tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("tls dial failed: %w", err)
	}
	defer rawConn.Close()

	t := &http2.Transport{}
	clientConn, err := t.NewClientConn(rawConn)
	if err != nil {
		return nil, fmt.Errorf("h2 client conn failed: %w", err)
	}
	defer clientConn.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := clientConn.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("h2 roundtrip failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
