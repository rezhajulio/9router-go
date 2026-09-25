package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// executeCursorLegacy runs a turn against api2.cursor.sh
// (ChatService/StreamUnifiedChatWithTools) using the Cursor legacy protobuf body.
func executeCursorLegacy(w http.ResponseWriter, req *Request, bodyMap map[string]any, accessToken, machineID string, ghostMode bool) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	headers := cursorpkg.BuildCursorHeaders(accessToken, machineID, ghostMode)
	url := cursorChatBaseURL + cursorChatPath

	var tools []any
	if t, ok := bodyMap["tools"].([]any); ok {
		tools = t
	}
	rawMsgs, _ := bodyMap["messages"].([]any)
	effort, _ := bodyMap["reasoning_effort"].(string)

	model := req.ModelName
	if model == "" {
		if m, ok := bodyMap["model"].(string); ok {
			model = m
		}
	}
	model = stripCursorModelPrefix(model)

	// Claude Code CLI traffic is forced into agent mode (#643). The inbound
	// User-Agent travels on the request context: ConnData only carries the
	// connection's providerSpecificData, so reading it from there never matched.
	forceAgentMode := false
	if ua := strings.ToLower(handlerutil.GetUserAgent(ctx)); ua != "" {
		if strings.Contains(ua, "claude-cli") || strings.Contains(ua, "claude-code") || strings.Contains(ua, "claude code") {
			forceAgentMode = true
		}
	}

	framedBody := cursorpkg.GenerateLegacyCursorBody(rawMsgs, model, tools, effort, forceAgentMode)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(framedBody))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	client := req.Client
	if client == nil {
		client = &http.Client{Timeout: cursorHeaderTimeout}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		// Surface the transport failure instead of writing a body and returning
		// nil: fallback treats a nil return as success, so a dial error used to
		// end the attempt silently (no rotation, no retry, zero-token usage log).
		return fmt.Errorf("cursor legacy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
		message := fmt.Sprintf("[%d]: %s", resp.StatusCode, strings.ReplaceAll(string(errBytes), `"`, `'`))
		return &proxy.UpstreamError{
			StatusCode: resp.StatusCode,
			Body:       cursorErrorBody(message, "invalid_request_error"),
		}
	}

	isStream := req.IsStream
	if s, ok := bodyMap["stream"].(bool); ok {
		isStream = s
	}

	composerModel := cursorpkg.IsComposerModel(model)
	responseID := fmt.Sprintf("chatcmpl-cursor-%d", time.Now().UnixMilli())
	created := time.Now().Unix()

	w = newCursorRecorderWriter(w, req)

	if isStream {
		return streamCursorLegacy(w, req, resp.Body, model, composerModel, responseID, created)
	}
	return respondCursorLegacy(w, resp.Body, model, composerModel, responseID, created, req)
}

// streamCursorLegacy translates Codebar ChatService frames into OpenAI SSE chunks.
func streamCursorLegacy(w http.ResponseWriter, req *Request, body io.Reader, model string, composerModel bool, responseID string, created int64) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	var pending []byte
	var totalThinking string
	emittedVisible := 0
	emitted := false
	streamErr := ""
	framingErr := false
	completionChars := 0
	// Published on every exit path, error frames included.
	defer func() { recordCursorUsage(req, completionChars) }()

	// Tool calls are indexed by id: a repeated id streams further arguments for
	// the same OpenAI tool_call instead of claiming a new index (upstream
	// accumulates by id and assigns the index on first sight).
	toolIndexes := make(map[string]int)
	toolCalls := 0

	buf := make([]byte, 4096)
	for {
		n, rErr := body.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			var ok bool
			pending, ok = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
				parsed := cursorpkg.ExtractLegacyResponse(payload)
				if parsed.Error != "" {
					streamErr = parsed.Error
					return
				}
				if parsed.ToolCall != nil {
					tc := parsed.ToolCall
					index, seen := toolIndexes[tc.ID]
					if !seen {
						index = toolCalls
						toolIndexes[tc.ID] = index
						toolCalls++
					}
					completionChars += len(tc.Arguments)
					writeSSEToolCall(w, flusher, responseID, created, model, tc.ID, tc.Name, tc.Arguments, index)
					emitted = true
				}
				if parsed.Text != "" {
					completionChars += len(parsed.Text)
					writeSSEChunk(w, flusher, responseID, created, model, parsed.Text, nil, "")
					emitted = true
				}
				if composerModel && parsed.Thinking != "" {
					totalThinking += parsed.Thinking
					vis := cursorpkg.VisibleComposerContentFromThinking(totalThinking)
					if len(vis) > emittedVisible {
						delta := vis[emittedVisible:]
						emittedVisible = len(vis)
						completionChars += len(delta)
						writeSSEChunk(w, flusher, responseID, created, model, delta, nil, "")
						emitted = true
					}
				}
			})
			if !ok {
				streamErr = "Cursor ChatService frame exceeded the accepted size"
				framingErr = true
			}
		}
		if rErr != nil || streamErr != "" {
			break
		}
	}

	if streamErr != "" {
		if !emitted {
			// Nothing was written yet, so report an HTTP error instead of a
			// truncated 200 stream. A framing failure is our own protocol handling
			// (502); everything else here is an upstream error frame, which Cursor
			// uses for rate limits.
			if framingErr {
				return &proxy.UpstreamError{
					StatusCode: http.StatusBadGateway,
					Body:       cursorErrorBody(streamErr, "api_error"),
				}
			}
			return &proxy.UpstreamError{
				StatusCode: http.StatusTooManyRequests,
				Body:       cursorErrorBody(streamErr, "rate_limit_error"),
			}
		}
		writeSSEError(w, flusher, streamErr)
		return nil
	}

	// finish_reason must reflect the turn: a tool call ends it as tool_calls.
	writeSSEChunk(w, flusher, responseID, created, model, "", nil, finishReasonFor(toolCalls))
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// respondCursorLegacy collects a ChatService turn and writes one chat.completion.
func respondCursorLegacy(w http.ResponseWriter, body io.Reader, model string, composerModel bool, responseID string, created int64, req *Request) error {
	var totalContent string
	var totalThinking string
	var toolCalls []map[string]any
	toolCallsByID := make(map[string]map[string]any)

	var pending []byte
	streamErr := ""
	framingErr := false
	buf := make([]byte, 4096)
	for {
		n, rErr := body.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			var ok bool
			pending, ok = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
				parsed := cursorpkg.ExtractLegacyResponse(payload)
				if parsed.Error != "" {
					// Upstream stops reading at the first error frame, then still
					// returns the completion when content already arrived.
					streamErr = parsed.Error
					return
				}
				if parsed.ToolCall != nil {
					tc := parsed.ToolCall
					existing, seen := toolCallsByID[tc.ID]
					if seen {
						fn := existing["function"].(map[string]any)
						fn["arguments"] = fn["arguments"].(string) + tc.Arguments
					} else {
						entry := map[string]any{
							"id":   tc.ID,
							"type": "function",
							"function": map[string]any{
								"name":      tc.Name,
								"arguments": tc.Arguments,
							},
						}
						toolCallsByID[tc.ID] = entry
						// Ordered on first sight; further arguments are appended in
						// place, so no ordering information is needed.
						toolCalls = append(toolCalls, entry)
					}
				}
				if parsed.Text != "" {
					totalContent += parsed.Text
				}
				if parsed.Thinking != "" {
					totalThinking += parsed.Thinking
				}
			})
			if !ok {
				streamErr = "Cursor ChatService frame exceeded the accepted size"
				framingErr = true
			}
		}
		if rErr != nil || streamErr != "" {
			break
		}
	}

	if streamErr != "" && totalContent == "" && len(toolCalls) == 0 {
		if framingErr {
			return &proxy.UpstreamError{
				StatusCode: http.StatusBadGateway,
				Body:       cursorErrorBody(streamErr, "api_error"),
			}
		}
		return &proxy.UpstreamError{
			StatusCode: http.StatusTooManyRequests,
			Body:       cursorErrorBody(streamErr, "rate_limit_error"),
		}
	}

	if composerModel && totalContent == "" && totalThinking != "" {
		totalContent = cursorpkg.VisibleComposerContentFromThinking(totalThinking)
	}

	msg := map[string]any{
		"role":    "assistant",
		"content": totalContent,
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
	}

	respPayload := map[string]any{
		"id":      responseID,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       msg,
				"finish_reason": finishReasonFor(len(toolCalls)),
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     len(req.Body) / 4,
			"completion_tokens": len(totalContent) / 4,
			"total_tokens":      (len(req.Body) + len(totalContent)) / 4,
		},
	}

	recordCursorUsage(req, len(totalContent)+len(totalThinking))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(respPayload); err != nil {
		return &errCursorCommitted{err: err}
	}
	return nil
}
