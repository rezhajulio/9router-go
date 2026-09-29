package executor

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"

	"golang.org/x/net/http2"
)

// cursorDialTimeout bounds the TCP connect and TLS handshake for the
// AgentService h2 socket. Reading the response body is not bounded: an agent
// turn streams for as long as the model needs.
const cursorDialTimeout = 15 * time.Second

// cursorHeaderTimeout bounds how long we wait for response headers (only) before
// giving up on an AgentService that accepted the connection and went silent.
const cursorHeaderTimeout = 60 * time.Second

// cursorReadIdleTimeout and cursorPingTimeout give the agent stream a liveness
// check instead of an overall deadline: after this much read silence the
// transport sends an HTTP/2 PING, and tears the connection down when the peer
// fails to ACK within the ping timeout. Reading the stream is deliberately
// unbounded, so without this a black-holed connection (NAT idle timeout, dropped
// FIN, hung server) would block the read loop forever.
const (
	cursorReadIdleTimeout = 30 * time.Second
	cursorPingTimeout     = 15 * time.Second
)

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
	// Pipe writes block until the transport consumes the bytes, so the mutex is
	// only held long enough to read the state: Close (including the one
	// context.AfterFunc runs when the client disconnects) takes the same mutex to
	// tear the session down, and holding it across a blocked write would park
	// both goroutines forever, with no deadline left to break the cycle.
	s.mu.Lock()
	if s.closed || s.pw == nil {
		s.mu.Unlock()
		return fmt.Errorf("session closed")
	}
	pw := s.pw
	s.mu.Unlock()

	_, err := pw.Write(frame)
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
	if s.body != nil {
		_ = s.body.Close()
	} else if s.resp != nil && s.resp.Body != nil {
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

	transport := &http2.Transport{
		ReadIdleTimeout: cursorReadIdleTimeout,
		PingTimeout:     cursorPingTimeout,
	}
	clientConn, err := transport.NewClientConn(rawConn)
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
	session.body = proxy.NewStallReaderWithContext(ctx, resp.Body, 0, "cursor-agent")
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

// dialCursorH2 connects to host:port and completes the TLS handshake. The
// connection is routed through the client's proxy (HTTP CONNECT) when it has
// one, because AgentService is h2-only and this socket is opened by hand: a raw
// dial would bypass the connection's proxy pool and strictProxy for the traffic
// that carries the account's bearer token.
func dialCursorH2(ctx context.Context, client *http.Client, host, port string, tlsConfig *tls.Config) (net.Conn, error) {
	return cursorpkg.DialH2(ctx, client, host, port, tlsConfig, cursorDialTimeout)
}
