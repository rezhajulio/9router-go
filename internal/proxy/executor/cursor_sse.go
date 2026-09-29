package executor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SSE chunk writers shared by the agent and legacy relays. Kept together so the
// OpenAI wire shape (object, delta keys, terminal chunk, [DONE]) has exactly one
// definition across both Cursor protocols.

func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model, content string, toolCalls []map[string]any, finishReason string) {
	writeSSEChunkWithUsage(w, flusher, id, created, model, content, toolCalls, finishReason, nil)
}

func writeSSEChunkWithUsage(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model, content string, toolCalls []map[string]any, finishReason string, usage map[string]any) {
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
	if usage != nil {
		chunk["usage"] = usage
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

// agentTrailerErrorStatus maps a Connect end-stream error code onto the status
// the caller reports, mirroring upstream's createErrorResponse: only
// resource_exhausted is a rate limit. unauthenticated is the one code that has
// to become 401, because that is what the fallback layer's token refresh and
// account rotation key off; every other code is a request-level rejection and
// stays a plain 400 so it neither benches the account nor rotates it.
func agentTrailerErrorStatus(code string) (int, string) {
	switch code {
	case "resource_exhausted":
		return http.StatusTooManyRequests, "rate_limit_error"
	case "unauthenticated":
		return http.StatusUnauthorized, "authentication_error"
	default:
		return http.StatusBadRequest, "api_error"
	}
}

// stripCursorModelPrefix removes the "cursor/" style routing prefix.
func stripCursorModelPrefix(model string) string {
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		return model[idx+1:]
	}
	return model
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
