package executor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/http2"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

const (
	cursorAgentRunPath = "/agent.v1.AgentService/Run"
	cursorAgentEndpoint = "https://agent.api5.cursor.sh"
	cursorChatBaseURL   = "https://api2.cursor.sh"
	cursorChatPath      = "/aiserver.v1.ChatService/StreamUnifiedChatWithTools"
)

type agentSession struct {
	rawConn    net.Conn
	clientConn *http2.ClientConn
	pw         *io.PipeWriter
	resp       *http.Response
	readBuf    []byte
	mu         sync.Mutex
	closed     bool
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
	if len(s.readBuf) == 0 {
		s.readBuf = make([]byte, 4096)
	}
	n, err := s.resp.Body.Read(s.readBuf)
	if n > 0 {
		return s.readBuf[:n], nil
	}
	return nil, err
}

func openAgentHttp2Stream(ctx context.Context, endpointURL string, headers map[string]string) (*agentSession, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor endpoint url: %w", err)
	}

	host := u.Host
	port := "443"
	if strings.Contains(host, ":") {
		h, p, splitErr := net.SplitHostPort(host)
		if splitErr == nil {
			host = h
			port = p
		}
	}

	tlsConfig := &tls.Config{
		ServerName: host,
		NextProtos: []string{"h2"},
	}

	d := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, err := tls.DialWithDialer(d, "tcp", net.JoinHostPort(host, port), tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial cursor h2 endpoint: %w", err)
	}

	t := &http2.Transport{}
	clientConn, err := t.NewClientConn(rawConn)
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

	resp, err := clientConn.RoundTrip(req)
	if err != nil {
		_ = pw.Close()
		_ = clientConn.Close()
		return nil, fmt.Errorf("h2 roundtrip failed: %w", err)
	}

	return &agentSession{
		rawConn:    rawConn,
		clientConn: clientConn,
		pw:         pw,
		resp:       resp,
	}, nil
}

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

	if cursorpkg.IsAgentCapableRequest(bodyMap) {
		err := executeCursorAgent(w, req, bodyMap, accessToken, machineID, ghostMode)
		if err == nil {
			return nil
		}
		// If AgentService fails before sending headers, fall through to legacy ChatService
	}

	return executeCursorLegacy(w, req, bodyMap, accessToken, machineID, ghostMode)
}

func executeCursorAgent(w http.ResponseWriter, req *Request, bodyMap map[string]any, accessToken, machineID string, ghostMode bool) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// 60s hard timeout to prevent hung HTTP/2 sessions matching upstream HTTP2_TIMEOUT_MS
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	headers := cursorpkg.BuildCursorHeaders(accessToken, machineID, ghostMode)
	url := cursorAgentEndpoint + cursorAgentRunPath

	session, err := openAgentHttp2Stream(ctx, url, headers)
	if err != nil {
		return err
	}
	defer session.Close()

	// Cancel upstream session cleanly on context cancellation
	stop := context.AfterFunc(ctx, session.Close)
	defer stop()

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
	// Strip provider prefix e.g. "cursor/" or "cu/"
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		model = model[idx+1:]
	}

	rawMsgs, _ := bodyMap["messages"].([]any)
	runFrame := cursorpkg.BuildAgentRunFrame(rawMsgs, model, tools)
	if err := session.Write(runFrame); err != nil {
		return fmt.Errorf("failed to write run frame: %w", err)
	}

	if session.resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(session.resp.Body)
		http.Error(w, fmt.Sprintf(`{"error":{"message":"Cursor AgentService %d: %s","type":"api_error"}}`,
			session.resp.StatusCode, strings.ReplaceAll(string(errBody), `"`, `'`)), session.resp.StatusCode)
		return nil
	}

	isStream := req.IsStream
	if s, ok := bodyMap["stream"].(bool); ok {
		isStream = s
	}

	composerModel := cursorpkg.IsComposerModel(model)
	responseID := fmt.Sprintf("chatcmpl-msg_%d", time.Now().UnixMilli())
	created := time.Now().Unix()

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)

		var pending []byte
		var thinkingAcc string
		emittedVisible := 0
		finished := false
		emittedText := false

		for !finished {
			chunk, readErr := session.ReadChunk()
			if len(chunk) > 0 {
				pending = append(pending, chunk...)
				pending = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
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
											writeSSEChunk(w, flusher, responseID, created, model, delta, nil, "")
										}
									} else {
										// Non-composer models stream thinking as reasoning_content
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
									writeSSEChunk(w, flusher, responseID, created, model, fb, nil, "")
								}
							}
							finished = true
							writeSSEChunk(w, flusher, responseID, created, model, "", nil, "stop")
							_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
							if flusher != nil {
								flusher.Flush()
							}
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
						if execReq.Has(10) {
							_ = session.Write(cursorpkg.CreateRequestContextResponse(execReq))
						} else if execReq.Has(11) {
							mcp := cursorpkg.DecodeMcpArgs(execReq.Get(11)[0].Value)
							name := mcp.ToolName
							if name == "" {
								name = mcp.Name
							}
							if name != "" {
								tcID := mcp.ToolCallID
								if tcID == "" {
									tcID = fmt.Sprintf("call_%s", uuid.New().String())
								}
								argsJSON, _ := json.Marshal(mcp.Args)
								finished = true
								writeSSEToolCall(w, flusher, responseID, created, model, tcID, name, string(argsJSON))
								writeSSEChunk(w, flusher, responseID, created, model, "", nil, "tool_calls")
								_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
								if flusher != nil {
									flusher.Flush()
								}
							} else {
								rej := cursorpkg.RejectExecRequest(execReq)
								if rej != nil {
									_ = session.Write(rej)
								} else {
									finished = true
									writeSSEChunk(w, flusher, responseID, created, model, "", nil, "stop")
									_, _ = fmt.Fprint(w, "data: {\"error\":{\"message\":\"Cursor AgentService requested an unsupported IDE tool\",\"type\":\"api_error\"}}\n\n")
									_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
									if flusher != nil {
										flusher.Flush()
									}
								}
							}
						} else {
							rej := cursorpkg.RejectExecRequest(execReq)
							if rej != nil {
								_ = session.Write(rej)
							} else {
								finished = true
								writeSSEChunk(w, flusher, responseID, created, model, "", nil, "stop")
								_, _ = fmt.Fprint(w, "data: {\"error\":{\"message\":\"Cursor AgentService requested an unsupported IDE tool\",\"type\":\"api_error\"}}\n\n")
								_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
								if flusher != nil {
									flusher.Flush()
								}
							}
						}
					}
				})
			}
			if readErr != nil {
				break
			}
		}

		if !finished {
			if !emittedText && thinkingAcc != "" {
				fb := thinkingAcc
				if composerModel {
					fb = cursorpkg.VisibleComposerContentFromThinking(thinkingAcc)
				}
				if fb != "" {
					writeSSEChunk(w, flusher, responseID, created, model, fb, nil, "")
				}
			}
			writeSSEChunk(w, flusher, responseID, created, model, "", nil, "stop")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}

		return nil
	}

	// Non-streaming
	var content string
	var thinking string
	var toolCalls []map[string]any
	finishReason := "stop"
	var pending []byte
	finished := false

	for !finished {
		chunk, readErr := session.ReadChunk()
		if len(chunk) > 0 {
			pending = append(pending, chunk...)
			pending = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
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
					if execReq.Has(10) {
						_ = session.Write(cursorpkg.CreateRequestContextResponse(execReq))
					} else if execReq.Has(11) {
						mcp := cursorpkg.DecodeMcpArgs(execReq.Get(11)[0].Value)
						name := mcp.ToolName
						if name == "" {
							name = mcp.Name
						}
						if name != "" {
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
							finishReason = "tool_calls"
							finished = true
						} else {
							rej := cursorpkg.RejectExecRequest(execReq)
							if rej != nil {
								_ = session.Write(rej)
							} else {
								agentErr := "Cursor AgentService requested an unsupported IDE tool"
								http.Error(w, fmt.Sprintf(`{"error":{"message":%q,"type":"api_error"}}`, agentErr), http.StatusBadRequest)
								finished = true
								return
							}
						}
					} else {
						rej := cursorpkg.RejectExecRequest(execReq)
						if rej != nil {
							_ = session.Write(rej)
						} else {
							agentErr := "Cursor AgentService requested an unsupported IDE tool"
							http.Error(w, fmt.Sprintf(`{"error":{"message":%q,"type":"api_error"}}`, agentErr), http.StatusBadRequest)
							finished = true
							return
						}
					}
				}
			})
		}
		if readErr != nil {
			break
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

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(respPayload)
}

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
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		model = model[idx+1:]
	}

	// Detect Claude Code User-Agent to force Agent mode (#643)
	forceAgentMode := false
	if req.ConnData != nil {
		if ua, ok := req.ConnData["user-agent"].(string); ok {
			uaLower := strings.ToLower(ua)
			if strings.Contains(uaLower, "claude-cli") || strings.Contains(uaLower, "claude-code") {
				forceAgentMode = true
			}
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
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"message":"%s","type":"connection_error"}}`, err.Error()), http.StatusBadGateway)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf(`{"error":{"message":"[%d]: %s","type":"api_error"}}`,
			resp.StatusCode, strings.ReplaceAll(string(errBytes), `"`, `'`)), resp.StatusCode)
		return nil
	}

	isStream := req.IsStream
	if s, ok := bodyMap["stream"].(bool); ok {
		isStream = s
	}

	composerModel := cursorpkg.IsComposerModel(model)
	responseID := fmt.Sprintf("chatcmpl-cursor-%d", time.Now().UnixMilli())
	created := time.Now().Unix()

	if isStream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)

		var pending []byte
		var totalThinking string
		emittedVisible := 0

		buf := make([]byte, 4096)
		for {
			n, rErr := resp.Body.Read(buf)
			if n > 0 {
				pending = append(pending, buf[:n]...)
				pending = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
					parsed := cursorpkg.ExtractLegacyResponse(payload)
					if parsed.Error != "" {
						writeSSEChunk(w, flusher, responseID, created, model, "", nil, "stop")
						_, _ = fmt.Fprintf(w, "data: {\"error\":{\"message\":%q,\"type\":\"api_error\"}}\n\n", parsed.Error)
						_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
						if flusher != nil {
							flusher.Flush()
						}
						return
					}
					if parsed.ToolCall != nil {
						writeSSEToolCall(w, flusher, responseID, created, model, parsed.ToolCall.ID, parsed.ToolCall.Name, parsed.ToolCall.Arguments)
					}
					if parsed.Text != "" {
						writeSSEChunk(w, flusher, responseID, created, model, parsed.Text, nil, "")
					}
					if composerModel && parsed.Thinking != "" {
						totalThinking += parsed.Thinking
						vis := cursorpkg.VisibleComposerContentFromThinking(totalThinking)
						if len(vis) > emittedVisible {
							delta := vis[emittedVisible:]
							emittedVisible = len(vis)
							writeSSEChunk(w, flusher, responseID, created, model, delta, nil, "")
						}
					}
				})
			}
			if rErr != nil {
				break
			}
		}

		writeSSEChunk(w, flusher, responseID, created, model, "", nil, "stop")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	// Non-streaming
	var totalContent string
	var totalThinking string
	var toolCalls []map[string]any

	var pending []byte
	buf := make([]byte, 4096)
	for {
		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			pending = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
				parsed := cursorpkg.ExtractLegacyResponse(payload)
				if parsed.Error != "" {
					http.Error(w, fmt.Sprintf(`{"error":{"message":%q,"type":"api_error"}}`, parsed.Error), http.StatusBadRequest)
					return
				}
				if parsed.ToolCall != nil {
					toolCalls = append(toolCalls, map[string]any{
						"id":   parsed.ToolCall.ID,
						"type": "function",
						"function": map[string]any{
							"name":      parsed.ToolCall.Name,
							"arguments": parsed.ToolCall.Arguments,
						},
					})
				}
				if parsed.Text != "" {
					totalContent += parsed.Text
				}
				if parsed.Thinking != "" {
					totalThinking += parsed.Thinking
				}
			})
		}
		if rErr != nil {
			break
		}
	}

	if composerModel && totalContent == "" && totalThinking != "" {
		totalContent = cursorpkg.VisibleComposerContentFromThinking(totalThinking)
	}

	finishReason := "stop"
	msg := map[string]any{
		"role":    "assistant",
		"content": totalContent,
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
		finishReason = "tool_calls"
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
			"completion_tokens": len(totalContent) / 4,
			"total_tokens":      (len(req.Body) + len(totalContent)) / 4,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(respPayload)
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

func writeSSEToolCall(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model, tcID, name, args string) {
	tc := []map[string]any{
		{
			"index": 0,
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
