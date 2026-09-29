package executor

import (
	"io"
	"sync"
	"testing"

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
