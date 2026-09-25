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
	"golang.org/x/sync/singleflight"
)

const (
	cursorModelsEndpoint = "/agent.v1.AgentService/GetUsableModels"
	cursorModelsURL      = "https://agent.api5.cursor.sh" + cursorModelsEndpoint

	// cursorModelsDialTimeout bounds the TCP connect and TLS handshake. The
	// dial is context-aware, so a caller with a shorter deadline (the /v1/models
	// builder allows 3s) is not stuck waiting for this long.
	cursorModelsDialTimeout = 10 * time.Second
	// cursorModelsFetchTimeout bounds the call itself when no tighter deadline
	// comes from the caller.
	cursorModelsFetchTimeout = 10 * time.Second

	modelsCacheTTL   = 5 * time.Minute
	negativeCacheTTL = 30 * time.Second

	// maxCatalogCacheEntries bounds the catalog cache: one entry is kept per
	// (token, machineID) pair, so a proxy rotating through many Cursor
	// credentials would otherwise grow it forever.
	maxCatalogCacheEntries = 128
)

// FetchCursorProtoFunc allows mocking H2 proto fetch in tests.
var FetchCursorProtoFunc = fetchCursorProtoH2

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

	// modelsFlight collapses concurrent cache misses for the same credentials
	// into a single upstream fetch, so every /v1/models call does not open
	// another h2 connection while the first one is still in flight.
	modelsFlight singleflight.Group
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

// readCatalogCache returns the cached catalog for key when it has not expired.
func readCatalogCache(key string, now time.Time) ([]ModelEntry, bool) {
	catalogCacheMu.RLock()
	defer catalogCacheMu.RUnlock()
	cached, ok := catalogCache[key]
	if !ok || !cached.expiresAt.After(now) {
		return nil, false
	}
	return cached.models, true
}

func writeCatalogCache(key string, ttl time.Duration, models []ModelEntry) {
	now := time.Now()
	catalogCacheMu.Lock()
	defer catalogCacheMu.Unlock()
	if _, exists := catalogCache[key]; !exists && len(catalogCache) >= maxCatalogCacheEntries {
		evictCatalogCacheLocked(now)
	}
	catalogCache[key] = cachedCatalog{expiresAt: now.Add(ttl), models: models}
}

// evictCatalogCacheLocked drops expired entries and then arbitrary ones until
// the cache is under its cap (the same "hot or dead, never warm" reasoning as
// the rotating-proxy client cache). Caller holds catalogCacheMu.
func evictCatalogCacheLocked(now time.Time) {
	for k, cached := range catalogCache {
		if !cached.expiresAt.After(now) {
			delete(catalogCache, k)
		}
	}
	for k := range catalogCache {
		if len(catalogCache) < maxCatalogCacheEntries {
			return
		}
		delete(catalogCache, k)
	}
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
	if ctx == nil {
		ctx = context.Background()
	}

	key := catalogCacheKey(accessToken, machineID)
	if !forceRefresh {
		if models, ok := readCatalogCache(key, time.Now()); ok {
			return models, nil
		}
	}

	// Wait for the shared fetch through a channel so each caller still honours
	// its own deadline: a cancelled caller must not block on (or cancel) a fetch
	// another request started.
	// The fetch itself runs on a context that outlives the triggering caller
	// (singleflight has no cancellation propagation) and carries no other
	// request's cancellation; it is bounded by cursorModelsFetchTimeout.
	// A forced refresh must not join an in-flight non-forced fetch, or the caller
	// asking for fresh models would get a possibly stale result, so the flag is
	// part of the flight key (the cache key stays the same).
	flightKey := key
	if forceRefresh {
		flightKey += ":refresh"
	}
	result := modelsFlight.DoChan(flightKey, func() (any, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cursorModelsFetchTimeout)
		defer cancel()
		return fetchCursorModels(fetchCtx, key, accessToken, machineID, ghostMode)
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-result:
		if res.Err != nil {
			return nil, res.Err
		}
		if res.Val == nil {
			return nil, nil
		}
		return res.Val.([]ModelEntry), nil
	}
}

// fetchCursorModels performs the GetUsableModels call and updates the cache,
// including the short negative cache that stops a failing connection from
// hammering upstream on every /v1/models call.
func fetchCursorModels(ctx context.Context, key, accessToken, machineID string, ghostMode bool) ([]ModelEntry, error) {
	headers := BuildCursorHeaders(accessToken, machineID, ghostMode)
	headers["accept"] = "application/proto"
	headers["content-type"] = "application/proto"
	delete(headers, "connect-accept-encoding")
	delete(headers, "connect-protocol-version")

	payload, err := FetchCursorProtoFunc(ctx, cursorModelsURL, headers)
	if err != nil {
		writeCatalogCache(key, negativeCacheTTL, nil)
		return nil, err
	}

	models := ParseCursorUsableModels(payload)
	if len(models) == 0 {
		return nil, nil
	}
	writeCatalogCache(key, modelsCacheTTL, models)
	return models, nil
}

func fetchCursorProtoH2(ctx context.Context, endpointURL string, headers map[string]string) ([]byte, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, err
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}

	rawConn, err := dialCursorTLS(ctx, net.JoinHostPort(host, port), &tls.Config{
		ServerName: host,
		NextProtos: []string{"h2"},
	})
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

// dialCursorTLS connects and completes the TLS handshake. net.Dialer with
// tls.DialWithDialer ignores the caller's context entirely, which made a 3s
// /v1/models budget wait on the full 10s dial instead.
func dialCursorTLS(ctx context.Context, addr string, tlsConfig *tls.Config) (net.Conn, error) {
	d := &net.Dialer{Timeout: cursorModelsDialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}
