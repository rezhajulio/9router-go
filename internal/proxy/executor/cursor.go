package executor

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"9router/proxy/internal/proxy"
	"9router/proxy/internal/translator"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

const (
	cursorAgentRunPath = "/agent.v1.AgentService/Run"
	cursorChatPath     = "/aiserver.v1.ChatService/StreamUnifiedChatWithTools"
)

// ForwardCursor handles chat completion execution for Cursor IDE provider.
func ForwardCursor(w http.ResponseWriter, req *Request) error {
	var bodyMap map[string]any
	if err := json.Unmarshal(req.Body, &bodyMap); err != nil {
		http.Error(w, `{"error":{"message":"invalid JSON body"}}`, http.StatusBadRequest)
		return nil
	}

	accessToken := req.APIKey
	machineID := ""
	ghostMode := true
	if req.ConnData != nil {
		if m, ok := req.ConnData["machineId"].(string); ok {
			machineID = m
		}
		if g, ok := req.ConnData["ghostMode"].(bool); ok {
			ghostMode = g
		}
	}

	needsTranslation := req.TranslateResp || translator.NeedsResponsesBridge(req.Ctx)
	if !needsTranslation {
		return forwardCursorDirect(w, req, bodyMap, accessToken, machineID, ghostMode)
	}

	isStream := req.IsStream
	if s, ok := bodyMap["stream"].(bool); ok {
		isStream = s
	}

	subReq := *req
	subReq.ResponseBuf = nil
	subReq.TTFT = nil

	if !isStream {
		rec := httptest.NewRecorder()
		if err := forwardCursorDirect(rec, &subReq, bodyMap, accessToken, machineID, ghostMode); err != nil {
			return err
		}
		return jsonResponse(req.Ctx, w, rec.Body, req.TranslateResp, req.ResponseBuf)
	}

	tw := newTranslatingWriter(w, req)
	defer tw.Close()
	return forwardCursorDirect(tw, &subReq, bodyMap, accessToken, machineID, ghostMode)
}

func forwardCursorDirect(w http.ResponseWriter, req *Request, bodyMap map[string]any, accessToken, machineID string, ghostMode bool) error {
	if cursorpkg.IsAgentCapableRequest(bodyMap) {
		agentErr := executeCursorAgent(w, req, bodyMap, accessToken, machineID, ghostMode)
		if agentErr == nil {
			return nil
		}
		var committed *errCursorCommitted
		if errors.As(agentErr, &committed) {
			return committed
		}
		if cw, ok := w.(interface{ IsCommitted() bool }); ok && cw.IsCommitted() {
			return &errCursorCommitted{err: agentErr}
		}
		if req.Ctx != nil && req.Ctx.Err() != nil {
			return req.Ctx.Err()
		}

		agentUpstream := asUpstreamError(agentErr)
		if isAuthFailure(agentUpstream) || (agentUpstream != nil && agentUpstream.StatusCode == http.StatusTooManyRequests) {
			return agentUpstream
		}

		legacyErr := executeCursorLegacy(w, req, bodyMap, accessToken, machineID, ghostMode)
		if legacyErr == nil {
			return nil
		}

		legacyUpstream := asUpstreamError(legacyErr)
		if isAuthFailure(legacyUpstream) {
			return legacyUpstream
		}
		if isAuthFailure(agentUpstream) {
			return agentUpstream
		}
		if agentUpstream != nil {
			return agentUpstream
		}
		if legacyUpstream != nil {
			return legacyUpstream
		}
		return legacyErr
	}

	return executeCursorLegacy(w, req, bodyMap, accessToken, machineID, ghostMode)
}

// translatingWriter is the http.ResponseWriter handed to the Cursor executors
// when the client speaks a different wire format than OpenAI Chat Completions.
//
// The executors write OpenAI SSE into this writer as usual; the first byte
// starts a goroutine that reads the same bytes back out of an io.Pipe and
// pushes them through execSSEStream, which performs the Claude Messages or
// Responses translation and owns the real client writer. Header and Flush are
// therefore no-ops here — the translation goroutine is what talks to w.
type translatingWriter struct {
	w       http.ResponseWriter
	req     *Request
	header  http.Header
	started bool
	pr      *io.PipeReader
	pw      *io.PipeWriter
	errCh   chan error
	closed  bool
	mu      sync.Mutex
}

func newTranslatingWriter(w http.ResponseWriter, req *Request) *translatingWriter {
	return &translatingWriter{
		w:      w,
		req:    req,
		header: make(http.Header),
	}
}

func (t *translatingWriter) Header() http.Header {
	return t.header
}

// Write starts the translation goroutine on the first call. Nothing reaches the
// client until then, so a pre-commit upstream failure can still be reported to
// the caller instead of a committed, empty response.
func (t *translatingWriter) Write(b []byte) (int, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	if !t.started {
		t.started = true
		t.pr, t.pw = io.Pipe()
		t.errCh = make(chan error, 1)
		go func() {
			t.errCh <- execSSEStream(t.w, t.pr, t.req)
		}()
	}
	pw := t.pw
	t.mu.Unlock()
	return pw.Write(b)
}

// WriteHeader is a no-op: the status is chosen by the translation goroutine
// once it has a first byte to translate.
func (t *translatingWriter) WriteHeader(code int) {}

// Flush is a no-op for the same reason as WriteHeader.
func (t *translatingWriter) Flush() {}

// IsCommitted reports whether the client response can no longer be replaced.
// Once the translator has started, its own headers are already out.
func (t *translatingWriter) IsCommitted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return true
	}
	if cw, ok := t.w.(interface{ IsCommitted() bool }); ok {
		return cw.IsCommitted()
	}
	return false
}

// Close ends the stream and returns the translation error, if any. It is
// idempotent: closing an unstarted writer is a no-op, and a second Close returns
// the already-collected result instead of blocking on an empty channel.
func (t *translatingWriter) Close() error {
	t.mu.Lock()
	if t.closed || !t.started {
		t.closed = true
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	pw := t.pw
	errCh := t.errCh
	t.mu.Unlock()

	_ = pw.Close()
	return <-errCh
}

// cursorRecorderWriter mirrors everything written to the client into
// req.ResponseBuf and stamps TTFT on the first byte.
//
// Executors that build their own stream must record this themselves (see
// stream.go, claude_messages.go); Cursor did not, so every Cursor turn was
// logged with zero completion tokens and no TTFT.
type cursorRecorderWriter struct {
	http.ResponseWriter
	buf       io.Writer
	ttft      *int64
	startTime time.Time
}

func newCursorRecorderWriter(w http.ResponseWriter, req *Request) http.ResponseWriter {
	if req.ResponseBuf == nil && req.TTFT == nil {
		return w
	}
	return &cursorRecorderWriter{
		ResponseWriter: w,
		buf:            req.ResponseBuf,
		ttft:           req.TTFT,
		startTime:      req.StartTime,
	}
}

func (c *cursorRecorderWriter) Write(b []byte) (int, error) {
	if c.ttft != nil && *c.ttft == 0 && !c.startTime.IsZero() {
		*c.ttft = time.Since(c.startTime).Milliseconds()
	}
	if c.buf != nil {
		_, _ = c.buf.Write(b)
	}
	return c.ResponseWriter.Write(b)
}

// Flush keeps the wrapper usable as an http.Flusher, which the SSE writers
// assert on; without it streaming responses would only flush at the end.
func (c *cursorRecorderWriter) Flush() {
	if flusher, ok := c.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// recordCursorUsage publishes how much text the turn produced, so usage logging
// does not fall back to estimating tokens from the raw SSE/JSON bytes mirrored
// into ResponseBuf (JSON framing inflates that estimate several-fold). Matches
// the stream.go/openai.go pattern of setting usage on the request context.
func recordCursorUsage(req *Request, completionChars int) {
	if req == nil || req.Ctx == nil {
		return
	}
	translator.SetUsage(req.Ctx, &translator.OpenAIUsage{CompletionTokens: completionChars / 4})
}

// errCursorCommitted wraps a failure that happened after response bytes were
// written (typically the client disconnecting mid-body). The caller must not
// treat it as a retryable pre-commit failure.
type errCursorCommitted struct{ err error }

func (e *errCursorCommitted) Error() string {
	return "cursor: response already committed: " + e.err.Error()
}

func (e *errCursorCommitted) Unwrap() error { return e.err }

// isAuthFailure reports whether an upstream error means the credentials were
// rejected, which is what the token-refresh path reacts to.
func isAuthFailure(ue *proxy.UpstreamError) bool {
	return ue != nil && (ue.StatusCode == http.StatusUnauthorized || ue.StatusCode == http.StatusForbidden)
}

func asUpstreamError(err error) *proxy.UpstreamError {
	var ue *proxy.UpstreamError
	if errors.As(err, &ue) {
		return ue
	}
	return nil
}
