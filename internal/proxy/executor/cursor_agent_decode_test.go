package executor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// pipeSession returns a session whose upstream writes land in a buffer the test
// can inspect, plus a drainer so a blocking pipe write never wedges the call.
func pipeSession(t *testing.T) (*agentSession, func() [][]byte) {
	t.Helper()
	pr, pw := io.Pipe()
	session := &agentSession{pw: pw}

	var mu sync.Mutex
	var frames [][]byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			buf := make([]byte, 4096)
			n, err := pr.Read(buf)
			if n > 0 {
				mu.Lock()
				frames = append(frames, append([]byte(nil), buf[:n]...))
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	return session, func() [][]byte {
		_ = pw.Close()
		<-done
		mu.Lock()
		defer mu.Unlock()
		return append([][]byte(nil), frames...)
	}
}

// shellReadExecRequest builds an exec_request for a named IDE builtin variant.
func variantExecRequest(variant int) cursorpkg.DecodedMessage {
	return cursorpkg.DecodeMessage(cursorpkg.ConcatBuffers(
		cursorpkg.EncodeField(1, cursorpkg.WireVarint, 7),
		cursorpkg.EncodeField(15, cursorpkg.WireBytes, "exec-1"),
		cursorpkg.EncodeField(variant, cursorpkg.WireBytes, cursorpkg.ConcatBuffers(
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "ls -la"),
		)),
	))
}

// A variant the proxy declines with a typed rejection must be answered once: a
// rejection frame, never a rejection plus a throw. Sending both made the server
// read two contradictory answers for one exec id.
func TestHandleExecRequestAnsweredOnceWithTypedRejection(t *testing.T) {
	session, drain := pipeSession(t)

	_, ok := handleExecRequest(session, nil, variantExecRequest(7)) // 7 = read
	if ok {
		t.Fatalf("a declined builtin must not produce a client tool call")
	}
	frames := drain()
	if len(frames) != 1 {
		t.Fatalf("expected exactly 1 upstream answer, got %d", len(frames))
	}
	// The rejection carries an ExecClientMessage (field 2); a throw would carry
	// an ExecClientControlMessage (field 6 of the client message).
	msg := cursorpkg.DecodeMessage(frames[0][5:])
	if !msg.Has(2) {
		t.Fatalf("expected an ExecClientMessage rejection, got fields %v", msg.Keys())
	}
}

// A variant with no typed rejection shape gets the throw plus the matching
// stream close, which keeps the turn alive.
func TestHandleExecRequestUnmappedVariantThrowsAndCloses(t *testing.T) {
	session, drain := pipeSession(t)

	_, ok := handleExecRequest(session, nil, variantExecRequest(27))
	if ok {
		t.Fatalf("an unmapped builtin must not produce a client tool call")
	}
	frames := drain()
	if len(frames) != 2 {
		t.Fatalf("expected a throw and a stream close, got %d frames", len(frames))
	}
}

// A bridged builtin is emitted as a tool call and still answered upstream, so
// the model is not left waiting on the exec it asked for.
func TestHandleExecRequestBridgedBuiltinEmitsToolCall(t *testing.T) {
	session, drain := pipeSession(t)
	tools := []cursorpkg.ClientTool{{
		Name: "bash",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"command": map[string]any{"type": "string"}},
			"required":   []any{"command"},
		},
	}}

	call, ok := handleExecRequest(session, tools, variantExecRequest(2)) // 2 = shell
	if !ok {
		t.Fatalf("expected a bridged tool call")
	}
	if call.Name != "bash" || call.Args["command"] != "ls -la" {
		t.Fatalf("unexpected bridged call: %+v", call)
	}
	if got := agentToolCallJSON(call); got != `{"command":"ls -la"}` {
		t.Fatalf("unexpected arguments JSON: %s", got)
	}
	if len(drain()) == 0 {
		t.Fatalf("the rejection must still go upstream")
	}
}

// A payload with turn_ended stops processing there, so nothing from the same
// payload can be written after the terminal.
func TestDecodeAgentPayloadStopsAtTurnEnded(t *testing.T) {
	session, drain := pipeSession(t)
	defer drain()

	payload := cursorpkg.ConcatBuffers(
		cursorpkg.EncodeField(1, cursorpkg.WireBytes,
			cursorpkg.EncodeField(14, cursorpkg.WireVarint, 1)), // turn_ended
		cursorpkg.EncodeField(2, cursorpkg.WireBytes, cursorpkg.ConcatBuffers(
			cursorpkg.EncodeField(1, cursorpkg.WireVarint, 3),
			cursorpkg.EncodeField(11, cursorpkg.WireBytes, cursorpkg.ConcatBuffers(
				cursorpkg.EncodeField(1, cursorpkg.WireBytes, "get_weather"),
			)),
		)),
	)

	events := decodeAgentPayload(session, nil, payload)
	if len(events) == 0 || events[len(events)-1].Kind != agentEventTurnEnded {
		t.Fatalf("expected the payload to end at turn_ended, got %+v", events)
	}
	for _, ev := range events {
		if ev.Kind == agentEventToolCall {
			t.Fatalf("a tool call after turn_ended must not be surfaced")
		}
	}
}

// A complete payload must expose its deltas in wire order.
func TestDecodeAgentPayloadOrder(t *testing.T) {
	session, drain := pipeSession(t)
	defer drain()

	payload := cursorpkg.EncodeField(1, cursorpkg.WireBytes, cursorpkg.ConcatBuffers(
		cursorpkg.EncodeField(cursorpkg.FieldTextDelta, cursorpkg.WireBytes,
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "hello")),
		cursorpkg.EncodeField(cursorpkg.FieldThinkingDelta, cursorpkg.WireBytes,
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "thinking")),
	))

	events := decodeAgentPayload(session, nil, payload)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %+v", events)
	}
	if events[0].Kind != agentEventText || events[0].Text != "hello" {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[1].Kind != agentEventThinking || events[1].Text != "thinking" {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
}

// trailerFrame builds a Connect end-stream frame (flag 0x02) whose payload is a
// JSON error object, which is how a quota or auth failure arrives on HTTP 200.
func trailerFrame(t *testing.T, body string) []byte {
	t.Helper()
	p := []byte(body)
	f := make([]byte, 5+len(p))
	f[0] = cursorpkg.CompressFlagTrailer
	binary.BigEndian.PutUint32(f[1:5], uint32(len(p)))
	copy(f[5:], p)
	return f
}

// An error trailer must become a typed upstream error on the agent path too.
// Decoding it as protobuf yields no fields, which is how a quota failure used to
// surface as a generic "stream closed before the turn ended" 502 that rotated no
// account and cooled nothing down.
func TestStreamCursorAgentTrailerError(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		wantStatus int
	}{
		{
			name:       "resource exhausted is a rate limit",
			payload:    `{"error":{"code":"resource_exhausted","message":"quota exceeded"}}`,
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "unauthenticated is a 401 so the token refreshes",
			payload:    `{"error":{"code":"unauthenticated","message":"token expired"}}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "anything else is a plain request error",
			payload:    `{"error":{"code":"permission_denied","message":"model not on your plan"}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, stream := range []bool{true, false} {
		mode := map[bool]string{true: "stream", false: "non-stream"}[stream]
		for _, tt := range tests {
			t.Run(mode+"/"+tt.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				session := &agentSession{body: io.NopCloser(bytes.NewReader(trailerFrame(t, tt.payload)))}

				var err error
				if stream {
					err = streamCursorAgent(rec, &Request{}, session, "gpt-5.6", false, "chatcmpl-test", 1, nil)
				} else {
					err = respondCursorAgent(rec, session, "gpt-5.6", false, "chatcmpl-test", 1, &Request{}, nil)
				}

				var ue *proxy.UpstreamError
				if !errors.As(err, &ue) {
					t.Fatalf("expected *proxy.UpstreamError, got %T (%v)", err, err)
				}
				if ue.StatusCode != tt.wantStatus {
					t.Fatalf("status = %d, want %d", ue.StatusCode, tt.wantStatus)
				}
			})
		}
	}
}

// Once bytes are on the wire the trailer error cannot be reported as a status:
// the client gets an SSE error terminal and the attempt must not be retried.
func TestStreamCursorAgentTrailerErrorAfterOutput(t *testing.T) {
	text := cursorpkg.WrapConnectRPCFrame(cursorpkg.EncodeField(1, cursorpkg.WireBytes,
		cursorpkg.EncodeField(cursorpkg.FieldTextDelta, cursorpkg.WireBytes,
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "partial"))))
	body := append(append([]byte(nil), text...),
		trailerFrame(t, `{"error":{"code":"resource_exhausted","message":"quota exceeded"}}`)...)

	rec := httptest.NewRecorder()
	session := &agentSession{body: io.NopCloser(bytes.NewReader(body))}
	err := streamCursorAgent(rec, &Request{}, session, "gpt-5.6", false, "chatcmpl-test", 1, nil)

	var committed *errCursorCommitted
	if !errors.As(err, &committed) {
		t.Fatalf("expected *errCursorCommitted so the attempt is not retried, got %T (%v)", err, err)
	}
	out := rec.Body.String()
	if !strings.Contains(out, `"error"`) {
		t.Fatalf("expected an SSE error frame, got %q", out)
	}
	if !strings.Contains(out, "[DONE]") {
		t.Fatalf("expected the stream to terminate, got %q", out)
	}
	if strings.Count(out, "data: [DONE]") != 1 {
		t.Fatalf("expected exactly one terminal, got %q", out)
	}
}

// A JSON error trailer must be recognised before any protobuf decoding.
func TestDecodeAgentPayloadErrorTrailer(t *testing.T) {
	session, drain := pipeSession(t)
	defer drain()

	events := decodeAgentPayload(session, nil, []byte(`{"error":{"code":"resource_exhausted","message":"quota exceeded"}}`))
	if len(events) != 1 || events[0].Kind != agentEventError {
		t.Fatalf("expected a single error event, got %+v", events)
	}
	if events[0].ErrCode != "resource_exhausted" || events[0].ErrMessage != "quota exceeded" {
		t.Fatalf("unexpected error event: %+v", events[0])
	}
}
