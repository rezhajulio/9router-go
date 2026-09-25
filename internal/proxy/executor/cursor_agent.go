package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// executeCursorAgent runs a turn against agent.api5.cursor.sh (AgentService),
// which is HTTP/2-only and speaks Connect-RPC bidi streaming.
func executeCursorAgent(w http.ResponseWriter, req *Request, bodyMap map[string]any, accessToken, machineID string, ghostMode bool) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	headers := cursorpkg.BuildCursorHeaders(accessToken, machineID, ghostMode)
	url := cursorAgentEndpoint + cursorAgentRunPath

	var tools []any
	if t, ok := bodyMap["tools"].([]any); ok {
		tools = t
	}

	model := req.ModelName
	if model == "" {
		if m, ok := bodyMap["model"].(string); ok {
			model = m
		}
	}
	model = stripCursorModelPrefix(model)

	rawMsgs, _ := bodyMap["messages"].([]any)
	runFrame := cursorpkg.BuildAgentRunFrame(rawMsgs, model, tools)

	session, err := openAgentHttp2Stream(ctx, url, headers, runFrame, req.Client)
	if err != nil {
		return err
	}
	defer session.Close()

	// Cancel the upstream session as soon as the client goes away. The stream
	// itself stays open on req.Ctx: an agent turn has no overall deadline.
	stop := context.AfterFunc(ctx, session.Close)
	defer stop()

	if session.resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(session.resp.Body, 1*1024*1024))
		message := fmt.Sprintf("Cursor AgentService %d: %s", session.resp.StatusCode,
			strings.ReplaceAll(string(errBody), `"`, `'`))
		// Report the upstream failure instead of writing a body: fallback needs
		// the status to refresh tokens and rotate combo accounts, and usage
		// logging needs to see the attempt fail.
		return &proxy.UpstreamError{
			StatusCode: session.resp.StatusCode,
			Body:       cursorErrorBody(message, "api_error"),
		}
	}

	isStream := req.IsStream
	if s, ok := bodyMap["stream"].(bool); ok {
		isStream = s
	}

	composerModel := cursorpkg.IsComposerModel(model)
	responseID := fmt.Sprintf("chatcmpl-msg_%d", time.Now().UnixMilli())
	created := time.Now().Unix()

	w = newCursorRecorderWriter(w, req)

	if isStream {
		return streamCursorAgent(w, req, session, model, composerModel, responseID, created)
	}
	return respondCursorAgent(w, session, model, composerModel, responseID, created, req)
}

// streamCursorAgent relays an AgentService turn as OpenAI SSE chunks.
func streamCursorAgent(w http.ResponseWriter, req *Request, session *agentSession, model string, composerModel bool, responseID string, created int64) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	var pending []byte
	var thinkingAcc string
	emittedVisible := 0
	finished := false
	emittedText := false
	streamErr := ""
	toolIndex := 0
	completionChars := 0
	// Published on every exit path, error frames included.
	defer func() { recordCursorUsage(req, completionChars) }()

	for !finished && streamErr == "" {
		chunk, readErr := session.ReadChunk()
		if len(chunk) > 0 {
			pending = append(pending, chunk...)
			var ok bool
			pending, ok = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
				if finished {
					return
				}
				serverMsg := cursorpkg.DecodeMessage(payload)

				// interaction_update (field 1)
				if serverMsg.Has(cursorpkg.FieldInteractionUpdate) {
					update := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldInteractionUpdate)[0].Value)
					if update.Has(cursorpkg.FieldTextDelta) {
						sub := cursorpkg.DecodeMessage(update.Get(cursorpkg.FieldTextDelta)[0].Value)
						if sub.Has(1) {
							delta := string(sub.Get(1)[0].Value)
							if delta != "" {
								emittedText = true
								completionChars += len(delta)
								writeSSEChunk(w, flusher, responseID, created, model, delta, nil, "")
							}
						}
					}

					if update.Has(cursorpkg.FieldThinkingDelta) {
						sub := cursorpkg.DecodeMessage(update.Get(cursorpkg.FieldThinkingDelta)[0].Value)
						if sub.Has(1) {
							td := string(sub.Get(1)[0].Value)
							if td != "" {
								thinkingAcc += td
								if composerModel {
									vis := cursorpkg.VisibleComposerContentFromThinking(thinkingAcc)
									if len(vis) > emittedVisible {
										delta := vis[emittedVisible:]
										emittedVisible = len(vis)
										emittedText = true
										completionChars += len(delta)
										writeSSEChunk(w, flusher, responseID, created, model, delta, nil, "")
									}
								} else {
									// Non-composer models stream thinking as reasoning_content
									completionChars += len(td)
									writeSSEReasoningChunk(w, flusher, responseID, created, model, td)
								}
							}
						}
					}

					if update.Has(cursorpkg.FieldTurnEnded) {
						if !emittedText && thinkingAcc != "" {
							fb := thinkingAcc
							if composerModel {
								fb = cursorpkg.VisibleComposerContentFromThinking(thinkingAcc)
							}
							if fb != "" {
								completionChars += len(fb)
								writeSSEChunk(w, flusher, responseID, created, model, fb, nil, "")
							}
						}
						finished = true
						writeSSEChunk(w, flusher, responseID, created, model, "", nil, finishReasonFor(toolIndex))
						_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
						if flusher != nil {
							flusher.Flush()
						}
						// The turn is over and its terminal chunk is out: nothing else
						// in this payload may be handled, or a tool call from the same
						// frame would be written after [DONE].
						return
					}
				}

				// kv_server_message (field 4)
				if serverMsg.Has(cursorpkg.FieldKvServerMessage) {
					kv := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldKvServerMessage)[0].Value)
					var kvID uint64
					if kv.Has(1) {
						kvID = kv.Get(1)[0].Varint
					}
					var meta []byte
					if kv.Has(4) {
						meta = kv.Get(4)[0].Value
					}
					if kv.Has(2) {
						_ = session.Write(cursorpkg.EncodeKvClientMessage(kvID, 2, cursorpkg.EncodeField(1, cursorpkg.WireBytes, []byte{}), meta))
					} else if kv.Has(3) {
						_ = session.Write(cursorpkg.EncodeKvClientMessage(kvID, 3, []byte{}, meta))
					}
				}

				// exec_request (field 2)
				if serverMsg.Has(cursorpkg.FieldExecRequest) {
					execReq := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldExecRequest)[0].Value)
					switch {
					case execReq.Has(10):
						_ = session.Write(cursorpkg.CreateRequestContextResponse(execReq))
					case execReq.Has(11):
						mcp := cursorpkg.DecodeMcpArgs(execReq.Get(11)[0].Value)
						name := mcp.ToolName
						if name == "" {
							name = mcp.Name
						}
						if name == "" {
							// No name means nothing can be handed to the client;
							// decline the tool so the server keeps the turn alive
							// instead of the stream failing.
							writeExecControlFrames(session, execReq, "Cursor AgentService requested an MCP tool without a name", "exec_variant_unsupported")
							return
						}
						tcID := mcp.ToolCallID
						if tcID == "" {
							tcID = fmt.Sprintf("call_%s", uuid.New().String())
						}
						argsJSON, _ := json.Marshal(mcp.Args)
						// One server turn may request several tools, so each call is
						// emitted with its own index and the turn stays open until
						// the server ends it; finish_reason is decided there.
						completionChars += len(argsJSON)
						writeSSEToolCall(w, flusher, responseID, created, model, tcID, name, string(argsJSON), toolIndex)
						toolIndex++
					default:
						if rej := cursorpkg.RejectExecRequest(execReq); rej != nil {
							_ = session.Write(rej)
						} else {
							// A Cursor IDE builtin this proxy cannot execute, or a
							// newer CLI variant without a known rejected shape.
							// Answering with a throw keeps the turn going (this is
							// how omp answers variants it has no handler for),
							// whereas ending the stream failed the whole request.
							writeExecControlFrames(session, execReq,
								fmt.Sprintf("Cursor AgentService requested IDE tool variant %d, which this proxy cannot execute", cursorpkg.ExecRequestVariant(execReq)),
								"exec_variant_unsupported")
						}
					}
				}
			})
			if !ok {
				streamErr = "Cursor AgentService frame exceeded the accepted size"
			}
		}
		if readErr != nil {
			break
		}
	}

	// finished means the turn_ended terminal (finish chunk + [DONE]) is already on
	// the wire, so nothing else may be written for this request — not an error
	// frame, and above all not a second [DONE].
	if finished {
		return nil
	}

	if streamErr != "" && !emittedText && thinkingAcc == "" && toolIndex == 0 {
		// Nothing was written yet, so this can still be reported as a failed
		// attempt instead of a truncated SSE stream the caller would log as a
		// success. The only streaming failure left here is a framing one (an
		// oversized frame), hence 502.
		return &proxy.UpstreamError{StatusCode: http.StatusBadGateway, Body: cursorErrorBody(streamErr, "api_error")}
	}
	if streamErr != "" {
		writeSSEError(w, flusher, streamErr)
		return nil
	}

	if !emittedText && thinkingAcc == "" && toolIndex == 0 {
		// The connection ended before the turn did and produced nothing: reporting
		// a finish_reason stop here would present a dead stream as a complete empty
		// answer and log the attempt as a success.
		return &proxy.UpstreamError{
			StatusCode: http.StatusBadGateway,
			Body:       cursorErrorBody("Cursor AgentService stream closed before the turn ended", "api_error"),
		}
	}

	if !emittedText && thinkingAcc != "" {
		fb := thinkingAcc
		if composerModel {
			fb = cursorpkg.VisibleComposerContentFromThinking(thinkingAcc)
		}
		if fb != "" {
			completionChars += len(fb)
			writeSSEChunk(w, flusher, responseID, created, model, fb, nil, "")
		}
	}
	writeSSEChunk(w, flusher, responseID, created, model, "", nil, finishReasonFor(toolIndex))
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	return nil
}

// writeExecControlFrames declines an IDE builtin: a throw naming the variant plus
// the matching stream close, so the server resumes the turn instead of the
// client having to kill the stream.
func writeExecControlFrames(session *agentSession, execReq cursorpkg.DecodedMessage, message, code string) {
	for _, frame := range cursorpkg.ExecClientControlFrames(execReq, message, code) {
		_ = session.Write(frame)
	}
}

// respondCursorAgent collects an AgentService turn and writes one chat.completion.
func respondCursorAgent(w http.ResponseWriter, session *agentSession, model string, composerModel bool, responseID string, created int64, req *Request) error {
	var content string
	var thinking string
	var toolCalls []map[string]any
	finishReason := "stop"
	var pending []byte
	finished := false
	agentErr := ""
	framingErr := false

	for !finished && agentErr == "" {
		chunk, readErr := session.ReadChunk()
		if len(chunk) > 0 {
			pending = append(pending, chunk...)
			var ok bool
			pending, ok = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
				if finished {
					return
				}
				serverMsg := cursorpkg.DecodeMessage(payload)

				if serverMsg.Has(cursorpkg.FieldInteractionUpdate) {
					update := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldInteractionUpdate)[0].Value)
					if update.Has(cursorpkg.FieldTextDelta) {
						sub := cursorpkg.DecodeMessage(update.Get(cursorpkg.FieldTextDelta)[0].Value)
						if sub.Has(1) {
							content += string(sub.Get(1)[0].Value)
						}
					}
					if update.Has(cursorpkg.FieldThinkingDelta) {
						sub := cursorpkg.DecodeMessage(update.Get(cursorpkg.FieldThinkingDelta)[0].Value)
						if sub.Has(1) {
							thinking += string(sub.Get(1)[0].Value)
						}
					}
					if update.Has(cursorpkg.FieldTurnEnded) {
						finished = true
					}
				}

				if serverMsg.Has(cursorpkg.FieldKvServerMessage) {
					kv := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldKvServerMessage)[0].Value)
					var kvID uint64
					if kv.Has(1) {
						kvID = kv.Get(1)[0].Varint
					}
					var meta []byte
					if kv.Has(4) {
						meta = kv.Get(4)[0].Value
					}
					if kv.Has(2) {
						_ = session.Write(cursorpkg.EncodeKvClientMessage(kvID, 2, cursorpkg.EncodeField(1, cursorpkg.WireBytes, []byte{}), meta))
					} else if kv.Has(3) {
						_ = session.Write(cursorpkg.EncodeKvClientMessage(kvID, 3, []byte{}, meta))
					}
				}

				if serverMsg.Has(cursorpkg.FieldExecRequest) {
					execReq := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldExecRequest)[0].Value)
					switch {
					case execReq.Has(10):
						_ = session.Write(cursorpkg.CreateRequestContextResponse(execReq))
					case execReq.Has(11):
						mcp := cursorpkg.DecodeMcpArgs(execReq.Get(11)[0].Value)
						name := mcp.ToolName
						if name == "" {
							name = mcp.Name
						}
						if name == "" {
							writeExecControlFrames(session, execReq, "Cursor AgentService requested an MCP tool without a name", "exec_variant_unsupported")
							return
						}
						tcID := mcp.ToolCallID
						if tcID == "" {
							tcID = fmt.Sprintf("call_%s", uuid.New().String())
						}
						argsJSON, _ := json.Marshal(mcp.Args)
						toolCalls = append(toolCalls, map[string]any{
							"id":   tcID,
							"type": "function",
							"function": map[string]any{
								"name":      name,
								"arguments": string(argsJSON),
							},
						})
						// Keep reading: one turn may request several tools, and
						// the turn ends when the server says so.
						finishReason = "tool_calls"
					default:
						if rej := cursorpkg.RejectExecRequest(execReq); rej != nil {
							_ = session.Write(rej)
						} else {
							writeExecControlFrames(session, execReq,
								fmt.Sprintf("Cursor AgentService requested IDE tool variant %d, which this proxy cannot execute", cursorpkg.ExecRequestVariant(execReq)),
								"exec_variant_unsupported")
						}
					}
				}
			})
			if !ok {
				agentErr = "Cursor AgentService frame exceeded the accepted size"
				framingErr = true
			}
		}
		if readErr != nil {
			break
		}
	}

	if agentErr != "" {
		status := http.StatusBadRequest
		if framingErr {
			status = http.StatusBadGateway
		}
		return &proxy.UpstreamError{StatusCode: status, Body: cursorErrorBody(agentErr, "api_error")}
	}
	if !finished && content == "" && thinking == "" && len(toolCalls) == 0 {
		return &proxy.UpstreamError{
			StatusCode: http.StatusBadGateway,
			Body:       cursorErrorBody("Cursor AgentService stream closed before the turn ended", "api_error"),
		}
	}

	finalContent := content
	var reasoningContent string
	if composerModel {
		if finalContent == "" && thinking != "" {
			finalContent = cursorpkg.VisibleComposerContentFromThinking(thinking)
		}
	} else if thinking != "" {
		reasoningContent = strings.TrimSpace(thinking)
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
