package executor

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"9router/proxy/internal/proxy"

	"golang.org/x/net/http2"
)

// The agent path opens its own h2 socket to AgentService, which cannot go
// through an http.Client directly; a connection pinned to a proxy pool must
// therefore be tunneled with CONNECT instead of dialed directly.
func TestDialCursorH2ViaConnectProxy(t *testing.T) {
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "through-tunnel")
	}))
	target.EnableHTTP2 = true
	target.StartTLS()
	defer target.Close()

	var connects int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			t.Errorf("expected CONNECT, got %s", r.Method)
			return
		}
		atomic.AddInt32(&connects, 1)

		upstream, err := net.Dial("tcp", r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer upstream.Close()

		clientConn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer clientConn.Close()

		if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
		go func() { _, _ = io.Copy(upstream, clientConn) }()
		_, _ = io.Copy(clientConn, upstream)
	}))
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse target url: %v", err)
	}

	conn, err := dialCursorH2(context.Background(), client, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true, // httptest self-signed certificate
	})
	if err != nil {
		t.Fatalf("dialCursorH2 through proxy: %v", err)
	}
	defer conn.Close()

	clientConn, err := (&http2.Transport{}).NewClientConn(conn)
	if err != nil {
		t.Fatalf("h2 client conn: %v", err)
	}
	defer clientConn.Close()

	req, err := http.NewRequest(http.MethodPost, target.URL, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := clientConn.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip through tunnel: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "through-tunnel" {
		t.Fatalf("expected body through tunnel, got %q", string(body))
	}
	if atomic.LoadInt32(&connects) == 0 {
		t.Fatalf("proxy never received a CONNECT")
	}
}

// Without a proxy the dialer must still connect directly (no regression for
// connections that do not use a proxy pool).
func TestDialCursorH2Direct(t *testing.T) {
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "direct")
	}))
	target.EnableHTTP2 = true
	target.StartTLS()
	defer target.Close()

	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse target url: %v", err)
	}

	conn, err := dialCursorH2(context.Background(), nil, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("dialCursorH2 direct: %v", err)
	}
	defer conn.Close()

	clientConn, err := (&http2.Transport{}).NewClientConn(conn)
	if err != nil {
		t.Fatalf("h2 client conn: %v", err)
	}
	defer clientConn.Close()

	req, err := http.NewRequest(http.MethodPost, target.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := clientConn.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "direct" {
		t.Fatalf("expected body direct, got %q", string(body))
	}
}

// A SOCKS5 proxy (e.g. from ALL_PROXY) cannot be spoken to with an HTTP CONNECT
// line. Failing keeps the request from leaking past the proxy, and the caller
// falls back to the legacy path, which goes through net/http and supports SOCKS5.
func TestDialCursorH2RejectsNonHTTPProxyScheme(t *testing.T) {
	proxyURL, err := url.Parse("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	_, err = dialCursorH2(context.Background(), client, "127.0.0.1", "443", &tls.Config{
		ServerName: "localhost",
		NextProtos: []string{"h2"},
	})
	if err == nil {
		t.Fatalf("expected an error for a socks5 proxy")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Fatalf("expected an unsupported scheme error, got %v", err)
	}
}

// newRefusingProxy starts a proxy that answers every CONNECT with 403, the shape
// an allowlisting sandbox proxy uses, and returns its URL.
func newRefusingProxy(t *testing.T) *url.URL {
	t.Helper()
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked-by-allowlist", http.StatusForbidden)
	}))
	t.Cleanup(proxy.Close)
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	return proxyURL
}

// newH2Target starts an h2-over-TLS server that echoes a fixed body and returns
// its parsed URL.
func newH2Target(t *testing.T, body string) *url.URL {
	t.Helper()
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	target.EnableHTTP2 = true
	target.StartTLS()
	t.Cleanup(target.Close)
	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse target url: %v", err)
	}
	return targetURL
}

// FallbackTransport clients (the default handler client) retry a refused proxy
// directly, so the hand-built AgentService socket must do the same. Otherwise
// agent-only models become unusable behind an allowlisting proxy while the
// legacy net/http path still works.
func TestDialCursorH2FallsBackDirectOnProxyRefusal(t *testing.T) {
	targetURL := newH2Target(t, "direct-after-refusal")
	proxyURL := newRefusingProxy(t)

	client := &http.Client{
		Transport: proxy.NewFallbackTransport(&http.Transport{Proxy: http.ProxyURL(proxyURL)}),
	}

	conn, err := dialCursorH2(context.Background(), client, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("expected a direct fallback after the proxy refusal, got %v", err)
	}
	defer conn.Close()

	clientConn, err := (&http2.Transport{}).NewClientConn(conn)
	if err != nil {
		t.Fatalf("h2 client conn: %v", err)
	}
	defer clientConn.Close()

	req, _ := http.NewRequest(http.MethodPost, targetURL.String(), strings.NewReader("hello"))
	resp, err := clientConn.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip after direct fallback: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "direct-after-refusal" {
		t.Fatalf("expected the origin body, got %q", string(body))
	}
}

// A proxy-pool client is a bare transport with no direct fallback: a refused
// tunnel must fail closed so credentials never leak past the configured proxy.
func TestDialCursorH2ProxyPoolRefusalFailsClosed(t *testing.T) {
	targetURL := newH2Target(t, "must-not-be-reached")
	proxyURL := newRefusingProxy(t)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	_, err := dialCursorH2(context.Background(), client, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
	})
	if err == nil {
		t.Fatalf("a proxy-pool refusal must not be dialed around")
	}
	if !strings.Contains(err.Error(), "CONNECT") {
		t.Fatalf("expected a CONNECT failure, got %v", err)
	}
}
