package executor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"9router/proxy/internal/proxy"
	"9router/proxy/internal/translator"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

func TestCursorExecutorRegistration(t *testing.T) {
	RegisterAll()
	exec := Get("cursor")
	if exec == nil {
		t.Fatalf("expected cursor executor to be registered")
	}
}

func TestCursorLegacyExtractAndThinking(t *testing.T) {
	// Frame 1: thinking with </think> OK
	thinkingMsg := cursorpkg.EncodeField(1, cursorpkg.WireBytes, "private thinking</think>OK")
	respField := cursorpkg.EncodeField(25, cursorpkg.WireBytes, thinkingMsg)
	respMsg := cursorpkg.EncodeField(2, cursorpkg.WireBytes, respField)
	frame := cursorpkg.WrapConnectRPCFrame(respMsg)

	if len(frame) <= 5 {
		t.Fatalf("frame too short")
	}
	parsed := cursorpkg.ExtractLegacyResponse(frame[5:])
	if parsed.Thinking != "private thinking</think>OK" {
		t.Fatalf("unexpected thinking: %q", parsed.Thinking)
	}

	content := cursorpkg.VisibleComposerContentFromThinking(parsed.Thinking)
	if content != "OK" {
		t.Fatalf("expected 'OK', got %q", content)
	}
}

func TestCursorLegacyToolCallFieldNumbers(t *testing.T) {
	// ClientSideToolV2Call fields: TOOL_ID=3, TOOL_NAME=9, TOOL_RAW_ARGS=10, TOOL_IS_LAST=11
	var tcFields [][]byte
	tcFields = append(tcFields, cursorpkg.EncodeField(3, cursorpkg.WireBytes, "call_abc123\nmc_model1"))
	tcFields = append(tcFields, cursorpkg.EncodeField(9, cursorpkg.WireBytes, "mcp_custom_bash"))
	tcFields = append(tcFields, cursorpkg.EncodeField(10, cursorpkg.WireBytes, `{"cmd":"ls"}`))
	tcFields = append(tcFields, cursorpkg.EncodeField(11, cursorpkg.WireVarint, 1))

	tcMsg := cursorpkg.ConcatBuffers(tcFields...)
	framePayload := cursorpkg.EncodeField(1, cursorpkg.WireBytes, tcMsg)
	frame := cursorpkg.WrapConnectRPCFrame(framePayload)

	parsed := cursorpkg.ExtractLegacyResponse(frame[5:])
	if parsed.ToolCall == nil {
		t.Fatalf("expected tool call to be parsed")
	}
	if parsed.ToolCall.ID != "call_abc123" {
		t.Fatalf("expected ID call_abc123, got %q", parsed.ToolCall.ID)
	}
	if parsed.ToolCall.Name != "mcp_custom_bash" {
		t.Fatalf("expected Name mcp_custom_bash, got %q", parsed.ToolCall.Name)
	}
	if parsed.ToolCall.Arguments != `{"cmd":"ls"}` {
		t.Fatalf("expected Arguments, got %q", parsed.ToolCall.Arguments)
	}
	if !parsed.ToolCall.IsLast {
		t.Fatalf("expected IsLast=true")
	}
}

func TestCursorAgentFrameDecode(t *testing.T) {
	// Text delta frame: InteractionUpdate (1) -> TextDelta (1) -> string (1)
	textDelta := cursorpkg.EncodeField(1, cursorpkg.WireBytes, "hello from agent")
	updatePart := cursorpkg.EncodeField(1, cursorpkg.WireBytes, textDelta)
	serverMsg := cursorpkg.EncodeField(1, cursorpkg.WireBytes, updatePart)
	frame := cursorpkg.WrapConnectRPCFrame(serverMsg)

	var receivedText string
	cursorpkg.DecodeAgentFrames(frame, func(payload []byte) {
		msg := cursorpkg.DecodeMessage(payload)
		if msg.Has(1) {
			update := cursorpkg.DecodeMessage(msg.Get(1)[0].Value)
			if update.Has(1) {
				sub := cursorpkg.DecodeMessage(update.Get(1)[0].Value)
				if sub.Has(1) {
					receivedText = string(sub.Get(1)[0].Value)
				}
			}
		}
	})

	if receivedText != "hello from agent" {
		t.Fatalf("expected 'hello from agent', got %q", receivedText)
	}
}

func TestForwardCursorInvalidJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := &Request{
		ModelName: "cursor/gpt-5.2",
		Body:      []byte("invalid json"),
	}

	err := ForwardCursor(rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestCursorAgentExecRequestHandling(t *testing.T) {
	// Build an execRequest with field 10 (request_context)
	execServerMsg := cursorpkg.EncodeField(10, cursorpkg.WireBytes, []byte{})
	agentServerMsg := cursorpkg.EncodeField(2, cursorpkg.WireBytes, execServerMsg)
	frame := cursorpkg.WrapConnectRPCFrame(agentServerMsg)

	var ackSent []byte
	cursorpkg.DecodeAgentFrames(frame, func(payload []byte) {
		msg := cursorpkg.DecodeMessage(payload)
		if msg.Has(2) {
			execReq := cursorpkg.DecodeMessage(msg.Get(2)[0].Value)
			if execReq.Has(10) {
				ackSent = cursorpkg.CreateRequestContextResponse(execReq)
			}
		}
	})

	if len(ackSent) == 0 {
		t.Fatalf("expected requestContext response to be generated")
	}
}

func TestCursorComposerThinkingNonStreaming(t *testing.T) {
	thinking := "reasoning text</think>Answer"
	vis := cursorpkg.VisibleComposerContentFromThinking(thinking)
	if vis != "Answer" {
		t.Fatalf("expected 'Answer', got %q", vis)
	}
}

func TestCursorLegacyServerResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// StreamUnifiedChatResponse (field 2) -> RESPONSE_TEXT (field 1).
		respMsg := cursorpkg.EncodeField(2, cursorpkg.WireBytes,
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "Hello world"))
		frame := cursorpkg.WrapConnectRPCFrame(respMsg)
		w.Header().Set("Content-Type", "application/connect+proto")
		_, _ = w.Write(frame)
	}))
	defer server.Close()

	restore := pointCursorLegacyAt(server.URL)
	defer restore()

	rec := httptest.NewRecorder()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
		"stream": false,
	}
	b, _ := json.Marshal(body)
	req := &Request{
		ModelName: "gpt-5.2",
		Body:      b,
		APIKey:    "test-token",
	}

	if err := executeCursorLegacy(rec, req, body, "test-token", "test-machine", true); err != nil {
		t.Fatalf("executeCursorLegacy failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &completion); err != nil {
		t.Fatalf("unmarshal completion: %v (body=%s)", err, rec.Body.String())
	}
	if len(completion.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(completion.Choices))
	}
	if completion.Choices[0].Message.Content != "Hello world" {
		t.Fatalf("expected content %q, got %q", "Hello world", completion.Choices[0].Message.Content)
	}
	if completion.Choices[0].FinishReason != "stop" {
		t.Fatalf("expected finish_reason stop, got %q", completion.Choices[0].FinishReason)
	}
}

// A non-200 from ChatService must reach the fallback layer as an error: returning
// nil made fallback record a success, so 401s never refreshed and combos never
// rotated.
func TestCursorLegacyUpstreamErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid token"}}`))
	}))
	defer server.Close()

	restore := pointCursorLegacyAt(server.URL)
	defer restore()

	rec := httptest.NewRecorder()
	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	b, _ := json.Marshal(body)
	req := &Request{ModelName: "gpt-5.2", Body: b, APIKey: "bad-token"}

	err := executeCursorLegacy(rec, req, body, "bad-token", "test-machine", true)
	var ue *proxy.UpstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *proxy.UpstreamError, got %T (%v)", err, err)
	}
	if ue.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", ue.StatusCode)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("executor must not write the body itself, got %q", rec.Body.String())
	}
}

// An error frame with no content must surface as an upstream error instead of a
// 200 with an empty completion.
func TestCursorLegacyErrorFrameWithoutContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		frame := cursorpkg.WrapConnectRPCFrame([]byte(`{"error":{"message":"resource_exhausted"}}`))
		_, _ = w.Write(frame)
	}))
	defer server.Close()

	restore := pointCursorLegacyAt(server.URL)
	defer restore()

	rec := httptest.NewRecorder()
	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	b, _ := json.Marshal(body)
	req := &Request{ModelName: "gpt-5.2", Body: b, APIKey: "test-token"}

	err := executeCursorLegacy(rec, req, body, "test-token", "test-machine", true)
	var ue *proxy.UpstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *proxy.UpstreamError, got %T (%v)", err, err)
	}
	if ue.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", ue.StatusCode)
	}
}

// A turn that emits a tool call must report finish_reason tool_calls, and the
// response must be written exactly once.
func TestCursorLegacyToolCallFinishReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tcFields [][]byte
		tcFields = append(tcFields, cursorpkg.EncodeField(3, cursorpkg.WireBytes, "call_1\nmc_x"))
		tcFields = append(tcFields, cursorpkg.EncodeField(9, cursorpkg.WireBytes, "mcp_custom_bash"))
		tcFields = append(tcFields, cursorpkg.EncodeField(10, cursorpkg.WireBytes, `{"cmd":"ls"}`))
		tcFields = append(tcFields, cursorpkg.EncodeField(11, cursorpkg.WireVarint, 1))
		tcMsg := cursorpkg.ConcatBuffers(tcFields...)
		_, _ = w.Write(cursorpkg.WrapConnectRPCFrame(cursorpkg.EncodeField(1, cursorpkg.WireBytes, tcMsg)))
	}))
	defer server.Close()

	restore := pointCursorLegacyAt(server.URL)
	defer restore()

	rec := httptest.NewRecorder()
	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	b, _ := json.Marshal(body)
	req := &Request{ModelName: "gpt-5.2", Body: b, APIKey: "test-token"}

	if err := executeCursorLegacy(rec, req, body, "test-token", "test-machine", true); err != nil {
		t.Fatalf("executeCursorLegacy failed: %v", err)
	}

	var completion struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &completion); err != nil {
		t.Fatalf("unmarshal completion: %v (body=%s)", err, rec.Body.String())
	}
	if len(completion.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(completion.Choices))
	}
	choice := completion.Choices[0]
	if choice.FinishReason != "tool_calls" {
		t.Fatalf("expected finish_reason tool_calls, got %q", choice.FinishReason)
	}
	if len(choice.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(choice.Message.ToolCalls))
	}
	if choice.Message.ToolCalls[0].ID != "call_1" {
		t.Fatalf("expected tool call id call_1, got %q", choice.Message.ToolCalls[0].ID)
	}
}

// pointCursorLegacyAt redirects the legacy ChatService endpoint at a test server.
// The live endpoint (api2.cursor.sh) must never be contacted from a unit test:
// with no proxy it dials the real host and blocks until the client timeout.
func pointCursorLegacyAt(baseURL string) func() {
	previous := cursorChatBaseURL
	cursorChatBaseURL = baseURL
	return func() { cursorChatBaseURL = previous }
}

// The turn must publish its own completion size. Without it, usage logging
// estimates tokens from the raw SSE/JSON bytes mirrored into ResponseBuf, which
// counts chunk ids and JSON framing and inflates the number several-fold.
func TestCursorLegacyPublishesUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respMsg := cursorpkg.EncodeField(2, cursorpkg.WireBytes,
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "Hello world"))
		_, _ = w.Write(cursorpkg.WrapConnectRPCFrame(respMsg))
	}))
	defer server.Close()

	restore := pointCursorLegacyAt(server.URL)
	defer restore()

	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "non-stream", true: "stream"}[stream], func(t *testing.T) {
			body := map[string]any{
				"messages": []any{map[string]any{"role": "user", "content": "hi"}},
				"stream":   stream,
			}
			b, _ := json.Marshal(body)
			ctx := translator.WithUsageCapture(context.Background())
			req := &Request{ModelName: "gpt-5.2", Body: b, APIKey: "test-token", IsStream: stream, Ctx: ctx}

			if err := executeCursorLegacy(httptest.NewRecorder(), req, body, "test-token", "test-machine", true); err != nil {
				t.Fatalf("executeCursorLegacy failed: %v", err)
			}

			usage := translator.GetAndClearUsage(ctx)
			if usage == nil {
				t.Fatalf("no usage published on the request context")
			}
			// "Hello world" is 11 chars -> 2 estimated tokens.
			if usage.CompletionTokens != 2 {
				t.Fatalf("CompletionTokens = %d, want 2", usage.CompletionTokens)
			}
		})
	}
}
