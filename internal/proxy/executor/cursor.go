package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	// The agent path opens its own HTTP/2 socket to AgentService, so it tunnels
	// through the connection's proxy with CONNECT (see dialCursorH2) rather than
	// silently dialing around a proxy pool.
	if cursorpkg.IsAgentCapableRequest(bodyMap) {
		agentErr := executeCursorAgent(w, req, bodyMap, accessToken, machineID, ghostMode)
		if agentErr == nil {
			return nil
		}
		// A failure after the response was committed (the client went away
		// mid-body) must not start a second upstream turn.
		var committed *errCursorCommitted
		if errors.As(agentErr, &committed) {
			return committed
		}
		// AgentService failed before writing anything: try the legacy ChatService.
		// Neither path returns nil-on-failure anymore, so a 401 here still reaches
		// the fallback layer (token refresh, combo rotation, usage logging).
		legacyErr := executeCursorLegacy(w, req, bodyMap, accessToken, machineID, ghostMode)
		if legacyErr == nil {
			return nil
		}

		// Report the most actionable status: an auth failure first (the refresh
		// path keys off 401/403), then the agent error (the primary path, whose
		// status reflects token validity), then the legacy error.
		agentUpstream, legacyUpstream := asUpstreamError(agentErr), asUpstreamError(legacyErr)
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

// clientProxyFor reports the proxy the request client would use for target, or
// nil for a direct connection. It inspects the transports the chat handler
// builds (a bare *http.Transport, possibly wrapped in FallbackTransport) so the
// decision is made from the resolved client rather than from connection config.
func clientProxyFor(client *http.Client, target string) *url.URL {
	if client == nil {
		return nil
	}
	rt := client.Transport
	for {
		switch t := rt.(type) {
		case *proxy.FallbackTransport:
			rt = t.Base
		case *http.Transport:
			if t.Proxy == nil {
				return nil
			}
			probe, err := http.NewRequest(http.MethodPost, target, nil)
			if err != nil {
				return nil
			}
			proxyURL, err := t.Proxy(probe)
			if err != nil {
				return nil
			}
			return proxyURL
		default:
			return nil
		}
	}
}

// cursorErrorBody renders the OpenAI-shaped error payload for a Cursor upstream
// failure. Fallback treats a returned error as a failed attempt, so the body
// only describes the failure to the caller; nothing is written to w.
func cursorErrorBody(message, errType string) []byte {
	body, err := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
			"code":    "",
		},
	})
	if err != nil {
		return []byte(`{"error":{"message":"cursor upstream error"}}`)
	}
	return body
}

func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model, content string, toolCalls []map[string]any, finishReason string) {
	delta := map[string]any{}
	if content != "" {
		delta["content"] = content
	}
	if len(toolCalls) > 0 {
		delta["tool_calls"] = toolCalls
	}

	chunk := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": delta,
			},
		},
	}
	if finishReason != "" {
		chunk["choices"].([]map[string]any)[0]["finish_reason"] = finishReason
	}

	b, _ := json.Marshal(chunk)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
	if flusher != nil {
		flusher.Flush()
	}
}

func writeSSEToolCall(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model, tcID, name, args string, index int) {
	tc := []map[string]any{
		{
			"index": index,
			"id":    tcID,
			"type":  "function",
			"function": map[string]any{
				"name":      name,
				"arguments": args,
			},
		},
	}
	writeSSEChunk(w, flusher, id, created, model, "", tc, "")
}

func writeSSEReasoningChunk(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model, reasoning string) {
	chunk := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"reasoning_content": reasoning,
				},
			},
		},
	}
	b, _ := json.Marshal(chunk)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
	if flusher != nil {
		flusher.Flush()
	}
}

// writeSSEError terminates a stream with an SSE error frame and [DONE]. A
// protocol failure is not a content delta: it must not be rendered as the
// assistant's reply, and no finish_reason chunk is emitted for it.
func writeSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	payload, err := json.Marshal(map[string]any{
		"error": map[string]any{"message": message, "type": "api_error"},
	})
	if err != nil {
		payload = []byte(`{"error":{"message":"cursor error","type":"api_error"}}`)
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// finishReasonFor reports the OpenAI finish_reason for a Cursor turn.
func finishReasonFor(toolCalls int) string {
	if toolCalls > 0 {
		return "tool_calls"
	}
	return "stop"
}

// stripCursorModelPrefix removes the "cursor/" style routing prefix.
func stripCursorModelPrefix(model string) string {
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		return model[idx+1:]
	}
	return model
}
