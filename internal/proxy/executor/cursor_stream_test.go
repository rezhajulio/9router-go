package executor

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// A turn that already wrote its terminal chunk must never write a second one,
// whatever the rest of the stream does: a second finish/[DONE] (or an error
// frame on top of a completed answer) double-terminates the SSE response and
// makes the caller treat a finished turn as a failed attempt worth retrying.
func TestStreamCursorAgentEmitsSingleTerminal(t *testing.T) {
	turnEnded := func() []byte {
		// interaction_update (1) -> turn_ended (14)
		return cursorpkg.EncodeField(1, cursorpkg.WireBytes,
			cursorpkg.EncodeField(14, cursorpkg.WireVarint, 1))
	}
	toolCall := func() []byte {
		mcp := cursorpkg.ConcatBuffers(
			cursorpkg.EncodeField(1, cursorpkg.WireBytes, "get_weather"),
			cursorpkg.EncodeField(3, cursorpkg.WireBytes, "call_1"),
			cursorpkg.EncodeField(5, cursorpkg.WireBytes, "get_weather"),
		)
		// exec_request (2) -> MCP tool call (11)
		return cursorpkg.EncodeField(2, cursorpkg.WireBytes,
			cursorpkg.EncodeField(11, cursorpkg.WireBytes, mcp))
	}

	tests := []struct {
		name  string
		chunk []byte
	}{
		{
			name:  "tool call in the same payload as turn end",
			chunk: cursorpkg.WrapConnectRPCFrame(cursorpkg.ConcatBuffers(turnEnded(), toolCall())),
		},
		{
			name: "oversized frame after turn end",
			chunk: append(cursorpkg.WrapConnectRPCFrame(turnEnded()),
				// 5-byte Connect header declaring 0x00900000 bytes, above
				// maxAgentFrameLength.
				0x00, 0x00, 0x90, 0x00, 0x00),
		},
		{
			name:  "stream ends without turn end and without output",
			chunk: cursorpkg.WrapConnectRPCFrame(cursorpkg.EncodeField(1, cursorpkg.WireBytes, []byte{})),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			session := &agentSession{body: io.NopCloser(bytes.NewReader(tt.chunk))}

			err := streamCursorAgent(rec, &Request{}, session, "gpt-5.6", false, "chatcmpl-test", 1, nil)

			if tt.name == "stream ends without turn end and without output" {
				// Nothing was produced: this must be reported as a failed attempt,
				// not as a complete empty answer.
				if err == nil {
					t.Fatalf("expected an error for a stream that produced nothing")
				}
				if rec.Body.Len() != 0 {
					t.Fatalf("nothing should be written before the failure, got %q", rec.Body.String())
				}
				return
			}

			if err != nil {
				t.Fatalf("streamCursorAgent returned %v, want nil for a completed turn", err)
			}
			body := rec.Body.String()
			if got := strings.Count(body, "data: [DONE]"); got != 1 {
				t.Fatalf("[DONE] emitted %d times, want exactly 1\n%s", got, body)
			}
			if got := strings.Count(body, `"finish_reason"`); got != 1 {
				t.Fatalf("finish_reason emitted %d times, want exactly 1\n%s", got, body)
			}
			if strings.Contains(body, `"error"`) {
				t.Fatalf("error frame written after the turn completed\n%s", body)
			}
			if strings.Contains(body, "tool_calls") {
				t.Fatalf("tool call written after the turn completed\n%s", body)
			}
		})
	}
}

// lockedRecorder is an httptest.ResponseRecorder safe for concurrent use: the
// relay loop writes from its own goroutine while the test polls the body.
type lockedRecorder struct {
	*httptest.ResponseRecorder
	mu sync.Mutex
}

func (l *lockedRecorder) Write(b []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ResponseRecorder.Write(b)
}

func (l *lockedRecorder) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ResponseRecorder.Flush()
}

func (l *lockedRecorder) snapshot() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ResponseRecorder.Body.String()
}

// A quiet upstream turn must not look stalled to the client: while AgentService
// sends nothing, the relay emits SSE comment keepalives so harnesses with a
// stream watchdog (Amp: "stream stalled after 120000 ms") keep the turn alive.
func TestStreamCursorAgentEmitsClientKeepalive(t *testing.T) {
	old := cursorClientPingInterval
	cursorClientPingInterval = 20 * time.Millisecond
	defer func() { cursorClientPingInterval = old }()

	pr, pw := io.Pipe()
	session := &agentSession{body: pr}
	rec := &lockedRecorder{ResponseRecorder: httptest.NewRecorder()}

	done := make(chan error, 1)
	go func() {
		done <- streamCursorAgent(rec, &Request{}, session, "gpt-5.6", false, "chatcmpl-test", 1, nil)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for {
		if strings.Contains(rec.snapshot(), ": ping\n\n") {
			break
		}
		if time.Now().After(deadline) {
			_ = pw.Close()
			t.Fatalf("no client keepalive within 3s of upstream silence\n%s", rec.snapshot())
		}
		time.Sleep(10 * time.Millisecond)
	}

	// End the turn cleanly afterwards: exactly one terminal chunk and [DONE].
	turnEnded := cursorpkg.WrapConnectRPCFrame(cursorpkg.EncodeField(1, cursorpkg.WireBytes,
		cursorpkg.EncodeField(14, cursorpkg.WireVarint, 1)))
	if _, err := pw.Write(turnEnded); err != nil {
		t.Fatalf("writing turn end: %v", err)
	}
	_ = pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("streamCursorAgent returned %v, want nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("streamCursorAgent did not return after turn end")
	}

	body := rec.snapshot()
	if got := strings.Count(body, "data: [DONE]"); got != 1 {
		t.Fatalf("[DONE] emitted %d times, want exactly 1\n%s", got, body)
	}
	if got := strings.Count(body, `"finish_reason"`); got != 1 {
		t.Fatalf("finish_reason emitted %d times, want exactly 1\n%s", got, body)
	}
}

var _ http.Flusher = (*lockedRecorder)(nil)
