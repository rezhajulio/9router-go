package cursor

import (
	"reflect"
	"strings"
	"testing"
)

func TestCursorChecksum(t *testing.T) {
	machineID := "test_machine_123"
	chk := GenerateCursorChecksum(machineID)
	if !strings.HasSuffix(chk, machineID) {
		t.Fatalf("checksum %q does not end with machineID %q", chk, machineID)
	}
	if len(chk) <= len(machineID) {
		t.Fatalf("checksum too short: %q", chk)
	}

	// Verify prefix matches Python / JS Jyh cipher base64 length (8 chars for 6 timestamp bytes)
	prefix := chk[:len(chk)-len(machineID)]
	if len(prefix) != 8 {
		t.Fatalf("expected 8 char prefix for 6 timestamp bytes, got %d (%q)", len(prefix), prefix)
	}
}

func TestAgentValueRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"null", nil},
		{"bool true", true},
		{"bool false", false},
		{"string", "hello world"},
		{"integer", float64(42)},
		{"float", 3.14},
		{"empty object", map[string]any{}},
		{"flat object", map[string]any{"a": float64(1), "b": "x", "c": true}},
		{"nested object", map[string]any{"outer": map[string]any{"inner": []any{float64(1), float64(2), "three"}}}},
		{"array mixed", []any{float64(1), "two", false, nil}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc := EncodeAgentValue(tc.value)
			dec := DecodeAgentValue(enc)

			if tc.value == nil {
				if dec != nil {
					t.Fatalf("expected nil, got %v", dec)
				}
				return
			}

			if !reflect.DeepEqual(dec, tc.value) {
				t.Fatalf("round-trip mismatch: got %v (%T), want %v (%T)", dec, dec, tc.value, tc.value)
			}
		})
	}
}

func TestMcpToolDefinition(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"city": map[string]any{"type": "string"},
		},
		"required": []any{"city"},
	}

	def := EncodeMcpToolDefinition("get_weather", "Get weather info", schema)
	msg := DecodeMessage(def)

	if !msg.Has(1) || string(msg.Get(1)[0].Value) != "get_weather" {
		t.Fatalf("name mismatch")
	}
	if !msg.Has(2) || string(msg.Get(2)[0].Value) != "Get weather info" {
		t.Fatalf("description mismatch")
	}
	if !msg.Has(4) || string(msg.Get(4)[0].Value) != "9router" {
		t.Fatalf("provider mismatch")
	}
	if !msg.Has(5) || string(msg.Get(5)[0].Value) != "get_weather" {
		t.Fatalf("tool_name mismatch")
	}

	decodedSchema := DecodeAgentValue(msg.Get(3)[0].Value)
	if !reflect.DeepEqual(decodedSchema, schema) {
		t.Fatalf("schema mismatch: got %v, want %v", decodedSchema, schema)
	}
}

func TestMcpArgsDecode(t *testing.T) {
	valBytes := EncodeAgentValue("Hanoi")
	entry := ConcatBuffers(
		EncodeField(1, WireBytes, "city"),
		EncodeField(2, WireBytes, valBytes),
	)

	mcpArgsBytes := ConcatBuffers(
		EncodeField(1, WireBytes, "get_weather"),
		EncodeField(2, WireBytes, entry),
		EncodeField(3, WireBytes, "call_1234"),
		EncodeField(5, WireBytes, "get_weather"),
	)

	res := DecodeMcpArgs(mcpArgsBytes)
	if res.Name != "get_weather" || res.ToolCallID != "call_1234" || res.ToolName != "get_weather" {
		t.Fatalf("unexpected McpArgs: %+v", res)
	}
	if res.Args["city"] != "Hanoi" {
		t.Fatalf("unexpected arg city: %v", res.Args["city"])
	}
}

func TestVisibleComposerContentFromThinking(t *testing.T) {
	thinking := "private reasoning that must not leak</think>OK"
	content := VisibleComposerContentFromThinking(thinking)
	if content != "OK" {
		t.Fatalf("expected 'OK', got %q", content)
	}

	noEnd := "private reasoning only"
	if res := VisibleComposerContentFromThinking(noEnd); res != "" {
		t.Fatalf("expected empty, got %q", res)
	}
}

func TestCursorUsableModelsParser(t *testing.T) {
	// Build mock GetUsableModelsResponse: repeated ModelDetails (field 1)
	m1 := ConcatBuffers(
		EncodeField(1, WireBytes, "default"),
		EncodeField(4, WireBytes, "Auto"),
	)
	m2 := ConcatBuffers(
		EncodeField(1, WireBytes, "gpt-5.3-codex"),
		EncodeField(4, WireBytes, "GPT 5.3 Codex"),
	)
	mDuplicate := ConcatBuffers(
		EncodeField(1, WireBytes, "gpt-5.3-codex"),
		EncodeField(4, WireBytes, "Duplicate"),
	)

	payload := ConcatBuffers(
		EncodeField(1, WireBytes, m1),
		EncodeField(1, WireBytes, m2),
		EncodeField(1, WireBytes, mDuplicate),
	)

	models := ParseCursorUsableModels(payload)
	if len(models) != 2 {
		t.Fatalf("expected 2 models after deduplication, got %d", len(models))
	}
	if models[0].ID != "default" || models[0].Name != "Auto" {
		t.Errorf("model 0 mismatch: %+v", models[0])
	}
	if models[1].ID != "gpt-5.3-codex" || models[1].Name != "GPT 5.3 Codex" {
		t.Errorf("model 1 mismatch: %+v", models[1])
	}
}

func TestBuildAgentRunFrame(t *testing.T) {
	msgs := []any{
		map[string]any{"role": "system", "content": "be brief"},
		map[string]any{"role": "user", "content": "hi"},
	}
	frame := BuildAgentRunFrame(msgs, "gpt-5.2", nil)
	if len(frame) <= 5 {
		t.Fatalf("frame too short: %d bytes", len(frame))
	}

	body := frame[5:]
	clientMsg := DecodeMessage(body)
	if !clientMsg.Has(1) {
		t.Fatalf("missing run_request")
	}

	run := DecodeMessage(clientMsg.Get(1)[0].Value)
	if !run.Has(2) {
		t.Fatalf("missing action")
	}
	if !run.Has(9) {
		t.Fatalf("missing requested_model")
	}
	if !run.Has(3) {
		t.Fatalf("missing ModelDetails")
	}
	if run.Has(8) {
		t.Fatalf("custom_system_prompt (field 8) must not be present")
	}
}

func TestRejectExecRequest(t *testing.T) {
	// Standard IDE builtin (field 2)
	req2 := ConcatBuffers(
		EncodeField(1, WireVarint, 1),
		EncodeField(2, WireBytes, "shell"),
	)
	msg2 := DecodeMessage(req2)
	rej2 := RejectExecRequest(msg2)
	if rej2 == nil {
		t.Fatalf("expected rejection frame for field 2")
	}

	// MCP server inspection / IDE exec with field 36 and context metadata 19 & 55
	req36 := ConcatBuffers(
		EncodeField(1, WireVarint, 1),
		EncodeField(19, WireBytes, "tracing_data"),
		EncodeField(36, WireBytes, EncodeField(1, WireBytes, "9router")),
		EncodeField(55, WireVarint, 0),
	)
	msg36 := DecodeMessage(req36)
	rej36 := RejectExecRequest(msg36)
	if rej36 == nil {
		t.Fatalf("expected rejection frame for field 36 with metadata")
	}

	// Verify rejected frame wraps Connect-RPC and ExecClientMessage (field 2)
	dec := DecodeMessage(rej36[5:])
	if !dec.Has(2) {
		t.Fatalf("expected ExecClientMessage (field 2)")
	}
	execClient := DecodeMessage(dec.Get(2)[0].Value)
	if !execClient.Has(36) {
		t.Fatalf("expected result payload under field 36")
	}
}

// TestBuildAgentRunFrameToolContinuation verifies that a normal tool turn
// (user, assistant tool_call, tool result) is not dropped: the tool call and
// its result must survive into history, and the current turn must be built
// from the trailing tool result (not the earlier user question) so the
// agent continues from what actually happened last.
func TestBuildAgentRunFrameToolContinuation(t *testing.T) {
	msgs := []any{
		map[string]any{"role": "user", "content": "run the tool"},
		map[string]any{
			"role":    "assistant",
			"content": "",
			"tool_calls": []any{
				map[string]any{
					"id":   "call_1",
					"type": "function",
					"function": map[string]any{
						"name":      "get_weather",
						"arguments": "{}",
					},
				},
			},
		},
		map[string]any{"role": "tool", "tool_call_id": "call_1", "content": "sunny, 22C"},
	}
	frame := BuildAgentRunFrame(msgs, "gpt-5.2", nil)
	body := frame[5:]
	clientMsg := DecodeMessage(body)
	run := DecodeMessage(clientMsg.Get(1)[0].Value)
	action := DecodeMessage(run.Get(2)[0].Value)
	userAction := DecodeMessage(action.Get(1)[0].Value)

	if !userAction.Has(1) {
		t.Fatalf("missing user message")
	}
	userMessage := DecodeMessage(userAction.Get(1)[0].Value)
	if !userMessage.Has(1) {
		t.Fatalf("missing user message text")
	}
	gotText := string(userMessage.Get(1)[0].Value)
	if gotText != "sunny, 22C" {
		t.Fatalf("expected current turn to be the tool result, got %q", gotText)
	}

	if !userAction.Has(7) {
		t.Fatalf("expected tool call + user question to survive into history")
	}
	history := DecodeMessage(userAction.Get(7)[0].Value)
	if len(history.Get(1)) != 2 {
		t.Fatalf("expected 2 history entries (user question + assistant tool_call), got %d", len(history.Get(1)))
	}
}

// An exec variant without a known rejected shape (newer Cursor CLI builds send
// 27-31, 36-38, 40-55) must be declined with a throw + stream close rather than
// ending the turn with "unsupported IDE tool".
func TestExecClientControlFrames(t *testing.T) {
	execReq := DecodeMessage(ConcatBuffers(
		EncodeField(1, WireVarint, 7),
		EncodeField(15, WireBytes, "exec-1"),
		EncodeField(27, WireBytes, []byte{}),
	))

	if got := ExecRequestVariant(execReq); got != 27 {
		t.Fatalf("expected variant 27, got %d", got)
	}
	if rej := RejectExecRequest(execReq); rej != nil {
		t.Fatalf("variant 27 has no known rejected shape, want nil, got %d bytes", len(rej))
	}
	// A known variant still gets the rejected result it always had.
	known := DecodeMessage(ConcatBuffers(
		EncodeField(1, WireVarint, 8),
		EncodeField(2, WireBytes, []byte{}),
	))
	if rej := RejectExecRequest(known); rej == nil {
		t.Fatalf("variant 2 must still be answered with a rejected result")
	}

	frames := ExecClientControlFrames(execReq, "cannot execute", "exec_variant_unsupported")
	if len(frames) != 2 {
		t.Fatalf("expected throw + stream close, got %d frames", len(frames))
	}

	var payloads [][]byte
	for _, f := range frames {
		DecodeAgentFrames(f, func(p []byte) {
			payloads = append(payloads, append([]byte(nil), p...))
		})
	}
	if len(payloads) != 2 {
		t.Fatalf("expected 2 decoded frames, got %d", len(payloads))
	}

	// AgentClientMessage.execClientControlMessage = 5, throw = 2.
	throwMsg := DecodeMessage(payloads[0])
	if !throwMsg.Has(5) {
		t.Fatalf("missing execClientControlMessage")
	}
	throw := DecodeMessage(DecodeMessage(throwMsg.Get(5)[0].Value).Get(2)[0].Value)
	if got := throw.Get(1)[0].Varint; got != 7 {
		t.Fatalf("throw id = %d, want 7", got)
	}
	if got := string(throw.Get(2)[0].Value); got != "cannot execute" {
		t.Fatalf("throw error = %q", got)
	}
	if got := string(throw.Get(4)[0].Value); got != "exec_variant_unsupported" {
		t.Fatalf("throw errorCode = %q", got)
	}

	// streamClose = 1.
	closeMsg := DecodeMessage(payloads[1])
	streamClose := DecodeMessage(DecodeMessage(closeMsg.Get(5)[0].Value).Get(1)[0].Value)
	if got := streamClose.Get(1)[0].Varint; got != 7 {
		t.Fatalf("streamClose id = %d, want 7", got)
	}
}
