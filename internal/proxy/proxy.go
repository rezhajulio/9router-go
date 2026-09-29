package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"9router/proxy/internal/constants"
)

// UpstreamError captures a non-200 upstream response.
type UpstreamError struct {
	StatusCode int
	Body       []byte
}

func (e *UpstreamError) Error() string {
	// Include the upstream body (truncated) so 4xx/5xx failures are diagnosable
	// from the fallback log alone — the body often carries Google/Antigravity's
	// actual rejection reason ("Invalid tool parameters", unknown model, etc.).
	body := strings.TrimSpace(string(e.Body))
	if strings.HasPrefix(body, "<!DOCTYPE html") || strings.HasPrefix(body, "<html") {
		lower := strings.ToLower(body)
		if strings.Contains(lower, "cloudflare") || strings.Contains(lower, "attention required") {
			return fmt.Sprintf("upstream returned %d: Cloudflare WAF challenge (Attention Required!): check User-Agent or network proxy", e.StatusCode)
		}
		if titleStart := strings.Index(lower, "<title>"); titleStart != -1 {
			titleEnd := strings.Index(lower[titleStart:], "</title>")
			if titleEnd != -1 {
				titleText := strings.TrimSpace(body[titleStart+7 : titleStart+titleEnd])
				return fmt.Sprintf("upstream returned %d: HTML page (%s)", e.StatusCode, titleText)
			}
		}
		return fmt.Sprintf("upstream returned %d: HTML error page", e.StatusCode)
	}
	if len(body) > 512 {
		body = body[:512] + "... (truncated)"
	}
	if body != "" {
		return fmt.Sprintf("upstream returned %d: %s", e.StatusCode, body)
	}
	return fmt.Sprintf("upstream returned %d", e.StatusCode)
}

var directProxyClient = &http.Client{
	Transport: constants.DefaultHTTPTransportConfig.NewTransport(),
}

// ClientProxyFor reports the proxy the client would use for target, or nil for a
// direct connection, plus whether that client retries directly on a proxy
// refusal. It inspects the transports the chat handler builds (a bare
// *http.Transport, possibly wrapped in FallbackTransport) so a hand-built socket
// can make the same routing decision the client itself would.
//
// allowDirect mirrors FallbackTransport, which backs the default client: an
// environment or sandbox proxy that refuses the tunnel is retried directly
// there, so a hand-rolled dialer has to behave the same way or its traffic fails
// behind such a proxy. A proxy-pool client is a bare transport and keeps failing
// closed, so credentials never leak past a configured proxy.
func ClientProxyFor(client *http.Client, target string) (proxyURL *url.URL, allowDirect bool) {
	if client == nil {
		return nil, false
	}
	rt := client.Transport
	for {
		switch t := rt.(type) {
		case *FallbackTransport:
			allowDirect = true
			rt = t.Base
		case *http.Transport:
			if t.Proxy == nil {
				return nil, allowDirect
			}
			probe, err := http.NewRequest(http.MethodPost, target, nil)
			if err != nil {
				return nil, allowDirect
			}
			resolved, err := t.Proxy(probe)
			if err != nil {
				return nil, allowDirect
			}
			return resolved, allowDirect
		default:
			return nil, allowDirect
		}
	}
}

func isProxyFailure(err error, resp *http.Response) bool {
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		if resp.Header.Get("X-Proxy-Error") != "" || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/plain") {
			return true
		}
	}
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "proxy") || strings.Contains(errStr, "connect tunnel failed") || strings.Contains(errStr, "blocked-by-allowlist") || strings.Contains(errStr, "forbidden") {
			return true
		}
	}
	return false
}

// DoRequest sends an HTTP POST to url with body and auth, returns the raw response.
// Caller must close resp.Body.
func DoRequest(ctx context.Context, client *http.Client, method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if isProxyFailure(err, resp) {
		if resp != nil {
			resp.Body.Close()
		}
		if directReq, dErr := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body)); dErr == nil {
			directReq.Header.Set("Content-Type", "application/json")
			for k, v := range headers {
				directReq.Header.Set(k, v)
			}
			if directResp, dErr2 := directProxyClient.Do(directReq); dErr2 == nil {
				resp = directResp
				err = nil
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("upstream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		errBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("upstream returned %d and body read failed: %w", resp.StatusCode, readErr)
		}
		return nil, &UpstreamError{StatusCode: resp.StatusCode, Body: errBody}
	}
	return resp, nil
}
