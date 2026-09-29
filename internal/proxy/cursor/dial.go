package cursor

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"9router/proxy/internal/proxy"
)

// DialTimeout bounds the TCP connect and each TLS handshake on a Cursor socket.
// Callers that need a different bound pass their own.
const DialTimeout = 15 * time.Second

// ErrProxyTunnel marks a failure to reach the proxy or establish the CONNECT
// tunnel, as opposed to a failure of the TLS handshake with the origin itself.
// Only the former justifies a direct retry: an origin-side handshake failure
// means the tunnel worked and the proxy is not the problem.
type ErrProxyTunnel struct{ Err error }

func (e *ErrProxyTunnel) Error() string { return "cursor proxy tunnel: " + e.Err.Error() }
func (e *ErrProxyTunnel) Unwrap() error { return e.Err }

// DialH2 connects to host:port and completes the TLS handshake, routing through
// the client's proxy when it has one. AgentService and the unary
// GetUsableModels call are HTTP/2-only and are opened by hand, so a raw dial
// would bypass the connection's proxy pool (and strictProxy) for traffic that
// carries the account's bearer token.
//
// A client backed by proxy.FallbackTransport retries directly after a proxy
// refusal, matching what the same request would have done through net/http. A
// bare proxy-pool transport keeps failing closed.
//
// timeout <= 0 selects DialTimeout.
func DialH2(ctx context.Context, client *http.Client, host, port string, tlsConfig *tls.Config, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		timeout = DialTimeout
	}
	addr := net.JoinHostPort(host, port)
	if proxyURL, allowDirect := proxy.ClientProxyFor(client, "https://"+addr); proxyURL != nil {
		switch proxyURL.Scheme {
		case "http", "https":
			conn, err := dialViaProxy(ctx, proxyURL, addr, tlsConfig, timeout)
			if err == nil {
				return conn, nil
			}
			var tunnel *ErrProxyTunnel
			if allowDirect && errors.As(err, &tunnel) {
				if direct, dErr := DialDirect(ctx, addr, tlsConfig, timeout); dErr == nil {
					return direct, nil
				}
			}
			return nil, err
		default:
			// Only HTTP CONNECT is implemented here; a SOCKS5 proxy (reachable via
			// ALL_PROXY) cannot be spoken to with a CONNECT line. Failing instead of
			// dialing direct keeps traffic from leaking past the proxy, and the
			// caller falls back to the legacy path, which goes through net/http and
			// supports SOCKS5 natively.
			return nil, fmt.Errorf("cursor: unsupported proxy scheme %q", proxyURL.Scheme)
		}
	}

	return DialDirect(ctx, addr, tlsConfig, timeout)
}

// DialDirect connects to addr and completes the TLS handshake bound to ctx,
// without going through any proxy.
func DialDirect(ctx context.Context, addr string, tlsConfig *tls.Config, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		timeout = DialTimeout
	}
	conn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.HandshakeContext(handshakeCtx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

// dialViaProxy tunnels through proxyURL with CONNECT and then runs the TLS
// handshake to addr inside the tunnel.
func dialViaProxy(ctx context.Context, proxyURL *url.URL, addr string, tlsConfig *tls.Config, timeout time.Duration) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if proxyURL.Port() == "" {
		port := "80"
		if proxyURL.Scheme == "https" {
			port = "443"
		}
		proxyAddr = net.JoinHostPort(proxyURL.Hostname(), port)
	}

	rawConn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, &ErrProxyTunnel{Err: fmt.Errorf("proxy dial %s: %w", proxyAddr, err)}
	}

	// The caller's context carries no deadline for a streaming turn, so the CONNECT
	// exchange and the inner TLS handshake need their own bound: a proxy that
	// accepts the TCP connection and then goes silent would otherwise hang before
	// the response-header timeout can even start.
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := rawConn.SetDeadline(deadline); err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("set tunnel deadline: %w", err)
	}

	conn := net.Conn(rawConn)
	if proxyURL.Scheme == "https" {
		proxyTLS := tls.Client(rawConn, &tls.Config{ServerName: proxyURL.Hostname()})
		if err := proxyTLS.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return nil, &ErrProxyTunnel{Err: fmt.Errorf("proxy tls handshake: %w", err)}
		}
		conn = proxyTLS
	}

	tunnelled, err := connectThroughProxy(conn, addr, proxyURL)
	if err != nil {
		_ = conn.Close()
		return nil, &ErrProxyTunnel{Err: err}
	}

	tlsConn := tls.Client(tunnelled, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = tunnelled.Close()
		return nil, fmt.Errorf("tls handshake through proxy: %w", err)
	}
	// The tunnel is up: the deadline covered establishing it, not the stream.
	if err := tlsConn.SetDeadline(time.Time{}); err != nil {
		_ = tunnelled.Close()
		return nil, fmt.Errorf("clear tunnel deadline: %w", err)
	}
	return tlsConn, nil
}

// connectThroughProxy issues "CONNECT addr" and returns a connection reading
// through the response reader, so bytes the proxy buffered after the response
// are not lost. Request writing and response parsing use net/http so header
// formatting and status handling match net/http's own proxy path.
func connectThroughProxy(conn net.Conn, addr string, proxyURL *url.URL) (net.Conn, error) {
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}
	if user := proxyURL.User; user != nil {
		password, _ := user.Password()
		credentials := base64.StdEncoding.EncodeToString([]byte(user.Username() + ":" + password))
		req.Header.Set("Proxy-Authorization", "Basic "+credentials)
	}
	if err := req.Write(conn); err != nil {
		return nil, fmt.Errorf("write CONNECT: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		return nil, fmt.Errorf("read CONNECT response: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy CONNECT %s failed: %s", addr, resp.Status)
	}
	return &bufferedConn{Conn: conn, reader: br}, nil
}

// bufferedConn keeps reads on the buffered reader used to parse the CONNECT
// response, so already-buffered tunnel bytes stay readable.
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(b []byte) (int, error) { return c.reader.Read(b) }
