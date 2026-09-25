package executor

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

// cursorDialTimeout bounds the TCP connect and TLS handshake for the
// AgentService h2 socket. Reading the response body is not bounded: an agent
// turn streams for as long as the model needs.
const cursorDialTimeout = 15 * time.Second

// cursorHeaderTimeout bounds how long we wait for response headers (only) before
// giving up on an AgentService that accepted the connection and went silent.
const cursorHeaderTimeout = 60 * time.Second

// AgentService is HTTP/2-only. These are variables so tests can point the
// executors at an httptest server instead of the live Cursor API.
var (
	cursorAgentEndpoint = "https://agent.api5.cursor.sh"
	cursorChatBaseURL   = "https://api2.cursor.sh"
)

type agentSession struct {
	rawConn    net.Conn
	clientConn *http2.ClientConn
	pw         *io.PipeWriter
	resp       *http.Response
	// body is captured separately from resp because Close can run concurrently
	// on context cancellation; ReadChunk reads through this field under the
	// mutex instead of touching resp.
	body    io.ReadCloser
	readBuf []byte
	mu      sync.Mutex
	closed  bool
}

func (s *agentSession) Write(frame []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.pw == nil {
		return fmt.Errorf("session closed")
	}
	_, err := s.pw.Write(frame)
	return err
}

func (s *agentSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.pw != nil {
		_ = s.pw.Close()
	}
	if s.resp != nil && s.resp.Body != nil {
		_ = s.resp.Body.Close()
	}
	if s.clientConn != nil {
		_ = s.clientConn.Close()
	}
	if s.rawConn != nil {
		_ = s.rawConn.Close()
	}
}

func (s *agentSession) ReadChunk() ([]byte, error) {
	// Take the body reference under the same mutex Close uses: on cancellation
	// Close tears the session down from another goroutine, and reading a body
	// that is being closed is what unblocks a pending read (it returns an
	// error). Touching s.resp directly here would race with that teardown.
	s.mu.Lock()
	body := s.body
	closed := s.closed
	s.mu.Unlock()
	if closed || body == nil {
		return nil, io.EOF
	}

	if len(s.readBuf) == 0 {
		s.readBuf = make([]byte, 4096)
	}
	n, err := body.Read(s.readBuf)
	if n > 0 {
		return s.readBuf[:n], nil
	}
	return nil, err
}

// openAgentHttp2Stream opens the AgentService bidi stream and writes
// initialFrame before waiting for the response headers. AgentService does not
// answer before it has the run request, so writing the frame from the goroutine
// that also waits for headers would stall until the header timeout expired.
func openAgentHttp2Stream(ctx context.Context, endpointURL string, headers map[string]string, initialFrame []byte, client *http.Client) (*agentSession, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor endpoint url: %w", err)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}

	rawConn, err := dialCursorH2(ctx, client, host, port, &tls.Config{
		ServerName: host,
		NextProtos: []string{"h2"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dial cursor h2 endpoint: %w", err)
	}

	clientConn, err := (&http2.Transport{}).NewClientConn(rawConn)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("failed to initialize h2 client conn: %w", err)
	}

	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, pr)
	if err != nil {
		_ = pw.Close()
		_ = clientConn.Close()
		return nil, fmt.Errorf("failed to build h2 request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	session := &agentSession{rawConn: rawConn, clientConn: clientConn, pw: pw}
	if len(initialFrame) > 0 {
		go func() {
			if _, writeErr := pw.Write(initialFrame); writeErr != nil {
				// Let RoundTrip fail instead of hanging on a body that can never
				// be written.
				_ = pw.CloseWithError(writeErr)
			}
		}()
	}

	resp, err := roundTripWithHeaderTimeout(ctx, clientConn, req)
	if err != nil {
		_ = pw.Close()
		_ = clientConn.Close()
		_ = rawConn.Close()
		return nil, err
	}
	session.resp = resp
	session.body = resp.Body
	return session, nil
}

// roundTripWithHeaderTimeout waits for response headers with a deadline, then
// leaves the stream open on ctx. The h2 RoundTrip goroutine is what actually
// carries the request body, so a timeout closes the connection to release it.
func roundTripWithHeaderTimeout(ctx context.Context, clientConn *http2.ClientConn, req *http.Request) (*http.Response, error) {
	type roundTripResult struct {
		resp *http.Response
		err  error
	}
	done := make(chan roundTripResult, 1)
	go func() {
		resp, err := clientConn.RoundTrip(req)
		done <- roundTripResult{resp: resp, err: err}
	}()

	timer := time.NewTimer(cursorHeaderTimeout)
	defer timer.Stop()

	select {
	case res := <-done:
		return res.resp, res.err
	case <-ctx.Done():
		_ = clientConn.Close()
		return nil, ctx.Err()
	case <-timer.C:
		_ = clientConn.Close()
		return nil, fmt.Errorf("cursor agent: no response headers after %s", cursorHeaderTimeout)
	}
}

// dialCursorH2 connects and completes the TLS handshake bound to ctx.
// net.Dialer with tls.DialWithDialer ignores the caller's context, so a
// cancelled client request used to keep a socket (and a goroutine) alive until
// the dialer timeout.
//
// AgentService is h2-only and this socket is opened by hand, so a raw dial would
// bypass the connection's proxy pool (and strictProxy) for the traffic that most
// text chats use. When the request client routes through a proxy the connection
// is therefore established with an HTTP CONNECT tunnel instead.
func dialCursorH2(ctx context.Context, client *http.Client, host, port string, tlsConfig *tls.Config) (net.Conn, error) {
	addr := net.JoinHostPort(host, port)
	if proxyURL := clientProxyFor(client, "https://"+addr); proxyURL != nil {
		return dialCursorH2ViaProxy(ctx, proxyURL, addr, tlsConfig)
	}

	conn, err := (&net.Dialer{Timeout: cursorDialTimeout}).DialContext(ctx, "tcp", addr)
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

// dialCursorH2ViaProxy tunnels through proxyURL with CONNECT and then runs the
// TLS handshake to addr inside the tunnel.
func dialCursorH2ViaProxy(ctx context.Context, proxyURL *url.URL, addr string, tlsConfig *tls.Config) (net.Conn, error) {
	proxyAddr := proxyURL.Host
	if proxyURL.Port() == "" {
		port := "80"
		if proxyURL.Scheme == "https" {
			port = "443"
		}
		proxyAddr = net.JoinHostPort(proxyURL.Hostname(), port)
	}

	rawConn, err := (&net.Dialer{Timeout: cursorDialTimeout}).DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("proxy dial %s: %w", proxyAddr, err)
	}

	conn := net.Conn(rawConn)
	if proxyURL.Scheme == "https" {
		proxyTLS := tls.Client(rawConn, &tls.Config{ServerName: proxyURL.Hostname()})
		if err := proxyTLS.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("proxy tls handshake: %w", err)
		}
		conn = proxyTLS
	}

	tunnelled, err := connectThroughProxy(conn, addr, proxyURL)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	tlsConn := tls.Client(tunnelled, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = tunnelled.Close()
		return nil, fmt.Errorf("tls handshake through proxy: %w", err)
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
