package cursor

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/net/http2"
)

// newCursorDialTarget starts an h2-over-TLS server that echoes body and returns
// its parsed URL.
func newCursorDialTarget(t *testing.T, body string) *url.URL {
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

// newConnectProxy starts a proxy that tunnels CONNECT to the requested host and
// counts the tunnels it established.
func newConnectProxy(t *testing.T) (*url.URL, *atomic.Int32) {
	t.Helper()
	var connects atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			t.Errorf("expected CONNECT, got %s", r.Method)
			return
		}
		connects.Add(1)
		upstream, err := net.Dial("tcp", r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		clientConn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer clientConn.Close()
		if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
		go func() { _, _ = io.Copy(upstream, clientConn) }()
		_, _ = io.Copy(clientConn, upstream)
	}))
	t.Cleanup(proxy.Close)
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	return proxyURL, &connects
}

func roundTrip(t *testing.T, conn net.Conn, target *url.URL) string {
	t.Helper()
	clientConn, err := (&http2.Transport{}).NewClientConn(conn)
	if err != nil {
		t.Fatalf("h2 client conn: %v", err)
	}
	defer clientConn.Close()
	req, _ := http.NewRequest(http.MethodPost, target.String(), strings.NewReader("hi"))
	resp, err := clientConn.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// DialH2 must route through the client's proxy: the model-discovery call carries
// the account's bearer token, so a raw dial would leak it past the pool.
func TestDialH2ThroughClientProxy(t *testing.T) {
	targetURL := newCursorDialTarget(t, "through-proxy")
	proxyURL, connects := newConnectProxy(t)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	conn, err := DialH2(context.Background(), client, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
	}, 0)
	if err != nil {
		t.Fatalf("DialH2 through proxy: %v", err)
	}
	defer conn.Close()

	if got := roundTrip(t, conn, targetURL); got != "through-proxy" {
		t.Fatalf("expected body through proxy, got %q", got)
	}
	if connects.Load() == 0 {
		t.Fatalf("proxy never received a CONNECT")
	}
}

// With no proxy configured the dialer connects directly.
func TestDialH2DirectWithoutProxy(t *testing.T) {
	targetURL := newCursorDialTarget(t, "direct-body")

	client := &http.Client{Transport: &http.Transport{}}
	conn, err := DialH2(context.Background(), client, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
	}, 0)
	if err != nil {
		t.Fatalf("DialH2 direct: %v", err)
	}
	defer conn.Close()

	if got := roundTrip(t, conn, targetURL); got != "direct-body" {
		t.Fatalf("expected direct body, got %q", got)
	}
}

// A bare proxy-pool transport has no direct fallback: a refused tunnel must fail
// closed so the credential cannot leak past the configured proxy.
func TestDialH2ProxyPoolRefusalFailsClosed(t *testing.T) {
	targetURL := newCursorDialTarget(t, "must-not-be-reached")
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked-by-allowlist", http.StatusForbidden)
	}))
	defer refusing.Close()
	proxyURL, _ := url.Parse(refusing.URL)

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	_, err := DialH2(context.Background(), client, targetURL.Hostname(), targetURL.Port(), &tls.Config{
		ServerName:         targetURL.Hostname(),
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true,
	}, 0)
	if err == nil {
		t.Fatalf("a proxy-pool refusal must not be dialed around")
	}
	var tunnel *ErrProxyTunnel
	if !errors.As(err, &tunnel) {
		t.Fatalf("expected an ErrProxyTunnel, got %T (%v)", err, err)
	}
}
