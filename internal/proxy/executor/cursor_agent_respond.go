package executor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// respondCursorAgent collects an AgentService turn and writes one chat.completion.
func respondCursorAgent(w http.ResponseWriter, session *agentSession, model string, composerModel bool, responseID string, created int64, req *Request, clientTools []cursorpkg.ClientTool) error {
	var content strings.Builder
	var thinking strings.Builder
	var toolCalls []map[string]any
	finishReason := "stop"
	var pending []byte
	finished := false
	agentErr := ""

	for !finished && agentErr == "" {
		chunk, readErr := session.ReadChunk()
		if len(chunk) > 0 {
			pending = append(pending, chunk...)
			var ok bool
			pending, ok = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
				if finished {
					return
				}
				for _, ev := range decodeAgentPayload(session, clientTools, payload) {
					switch ev.Kind {
					case agentEventText:
						content.WriteString(ev.Text)

					case agentEventThinking:
						thinking.WriteString(ev.Text)

					case agentEventToolCall:
						tcID := ev.ToolCall.ID
						if tcID == "" {
							tcID = fmt.Sprintf("call_%s", uuid.New().String())
						}
						toolCalls = append(toolCalls, map[string]any{
							"id":   tcID,
							"type": "function",
							"function": map[string]any{
								"name":      ev.ToolCall.Name,
								"arguments": agentToolCallJSON(ev.ToolCall),
							},
						})
						finishReason = "tool_calls"
						finished = true
						return

					case agentEventTurnEnded:
						finished = true
						return
					}
				}
			})
			if !ok {
				agentErr = "Cursor AgentService frame exceeded the accepted size"
			}
		}
		if readErr != nil {
			break
		}
	}
	if ctx := req.Ctx; ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if agentErr != "" {
		// The only terminal error this loop produces is a framing one (an
		// oversized frame), which is a protocol failure of our own handling
		// rather than anything the client asked for.
		return &proxy.UpstreamError{StatusCode: http.StatusBadGateway, Body: cursorErrorBody(agentErr, "api_error")}
	}
	finalContent := content.String()
	thinkingStr := thinking.String()
	if !finished && finalContent == "" && thinkingStr == "" && len(toolCalls) == 0 {
		return &proxy.UpstreamError{
			StatusCode: http.StatusBadGateway,
			Body:       cursorErrorBody("Cursor AgentService stream closed before the turn ended", "api_error"),
		}
	}

	var reasoningContent string
	if composerModel {
		if finalContent == "" && thinkingStr != "" {
			finalContent = cursorpkg.VisibleComposerContentFromThinking(thinkingStr)
		}
	} else if thinkingStr != "" {
		reasoningContent = strings.TrimSpace(thinkingStr)
	}

	msg := map[string]any{
		"role":    "assistant",
		"content": finalContent,
	}
	if reasoningContent != "" {
		msg["reasoning_content"] = reasoningContent
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
				"finish_reason": finishReason,
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     len(req.Body) / 4,
			"completion_tokens": len(finalContent) / 4,
			"total_tokens":      (len(req.Body) + len(finalContent)) / 4,
		},
	}

	recordCursorUsage(req, len(finalContent)+len(reasoningContent))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(respPayload); err != nil {
		return &errCursorCommitted{err: err}
	}
	return nil
}
