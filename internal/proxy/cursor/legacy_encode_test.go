package cursor

import (
	"strings"
	"testing"
)

// legacyRequestFields decodes a generated body down to the request message so a
// test can assert the fields Cursor actually receives.
func legacyRequestFields(t *testing.T, body []byte) DecodedMessage {
	t.Helper()
	if len(body) <= 5 {
		t.Fatalf("frame too short: %d bytes", len(body))
	}
	envelope := DecodeMessage(body[5:])
	if !envelope.Has(FieldLegacyRequest) {
		t.Fatalf("expected the request under field %d", FieldLegacyRequest)
	}
	return DecodeMessage(envelope.Get(FieldLegacyRequest)[0].Value)
}

// legacyMessageContents returns each conversation message's content field in
// order, skipping the non-message fields that share field number 1.
func legacyMessageContents(t *testing.T, req DecodedMessage) []string {
	t.Helper()
	var out []string
	for _, f := range req.Get(FieldLegacyMessages) {
		msg := DecodeMessage(f.Value)
		if !msg.Has(FieldLegacyMsgContent) || !msg.Has(FieldLegacyMsgRole) {
			continue // not a conversation message (different oneof arm)
		}
		out = append(out, string(msg.Get(FieldLegacyMsgContent)[0].Value))
	}
	return out
}

// A system message travels as a user bubble prefixed with the marker upstream's
// openai-to-cursor convertMessages writes, so the model can tell it from a real
// user turn.
func TestGenerateLegacyCursorBodySystemPrefix(t *testing.T) {
	body := GenerateLegacyCursorBody([]any{
		map[string]any{"role": "system", "content": "be terse"},
		map[string]any{"role": "user", "content": "hi"},
	}, "gpt-5.2", nil, "", false)

	got := legacyMessageContents(t, legacyRequestFields(t, body))
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d (%q)", len(got), got)
	}
	if got[0] != SystemInstructionPrefix+"be terse" {
		t.Fatalf("system message = %q, want the prefixed form", got[0])
	}
	if strings.HasPrefix(got[1], SystemInstructionPrefix) {
		t.Fatalf("a user message must not be prefixed: %q", got[1])
	}
}

// An OpenAI tool follow-up becomes a ToolResult pair carrying the name and
// arguments of the call it answers, which the tool message alone does not carry.
func TestGenerateLegacyCursorBodyToolResult(t *testing.T) {
	body := GenerateLegacyCursorBody([]any{
		map[string]any{"role": "user", "content": "run it"},
		map[string]any{
			"role": "assistant",
			"tool_calls": []any{map[string]any{
				"id":       "call_1",
				"type":     "function",
				"function": map[string]any{"name": "bash", "arguments": `{"command":"ls"}`},
			}},
		},
		map[string]any{"role": "tool", "tool_call_id": "call_1", "content": "a.txt"},
	}, "gpt-5.2", nil, "", false)

	req := legacyRequestFields(t, body)
	var found bool
	for _, f := range req.Get(FieldLegacyMessages) {
		msg := DecodeMessage(f.Value)
		if !msg.Has(FieldLegacyMsgToolResults) {
			continue
		}
		found = true
		tr := DecodeMessage(msg.Get(FieldLegacyMsgToolResults)[0].Value)
		if !tr.Has(FieldToolResultCallID) || string(tr.Get(FieldToolResultCallID)[0].Value) != "call_1" {
			t.Fatalf("tool result is missing the call id")
		}
		if !tr.Has(FieldToolResultName) || string(tr.Get(FieldToolResultName)[0].Value) != "mcp_custom_bash" {
			// formatToolName rewrites a bare OpenAI name into Cursor's
			// mcp_<server>_<tool> form, the same mapping upstream applies.
			t.Fatalf("tool result name = %q, want the mcp_-prefixed form",
				string(tr.Get(FieldToolResultName)[0].Value))
		}
		if !tr.Has(FieldToolResultRawArgs) || string(tr.Get(FieldToolResultRawArgs)[0].Value) != `{"command":"ls"}` {
			t.Fatalf("tool result is missing the call arguments")
		}
	}
	if !found {
		t.Fatalf("expected a tool result on the follow-up message")
	}
}

// Only the exact strings upstream recognises set the thinking level.
func TestGenerateLegacyCursorBodyThinkingLevel(t *testing.T) {
	thinkingLevel := func(effort string) uint64 {
		req := legacyRequestFields(t, GenerateLegacyCursorBody([]any{
			map[string]any{"role": "user", "content": "hi"},
		}, "gpt-5.2", nil, effort, false))
		if !req.Has(FieldLegacyThinkingLevel) {
			t.Fatalf("thinking level field missing for effort %q", effort)
		}
		return req.Get(FieldLegacyThinkingLevel)[0].Varint
	}

	for _, tc := range []struct {
		effort string
		want   uint64
	}{
		{"medium", 1},
		{"high", 2},
		{"max", 0},  // upstream sends UNSPECIFIED
		{"HIGH", 0}, // case-sensitive on purpose
		{"", 0},
		{"low", 0},
	} {
		if got := thinkingLevel(tc.effort); got != tc.want {
			t.Errorf("effort %q -> thinking level %d, want %d", tc.effort, got, tc.want)
		}
	}
}

// An empty name, description or schema must be omitted, matching upstream's
// encodeMcpTool, and a Claude-style `input_schema` must be accepted.
func TestGenerateLegacyCursorBodyMcpToolOmission(t *testing.T) {
	body := GenerateLegacyCursorBody([]any{
		map[string]any{"role": "user", "content": "hi"},
	}, "gpt-5.2", []any{
		map[string]any{"function": map[string]any{
			"name": "bare", // no description, no parameters
		}},
		map[string]any{
			"name":         "claude_style",
			"description":  "d",
			"input_schema": map[string]any{"type": "object"},
		},
	}, "", false)

	req := legacyRequestFields(t, body)
	tools := req.Get(FieldLegacyMcpTools)
	if len(tools) != 2 {
		t.Fatalf("expected 2 MCP tools, got %d", len(tools))
	}

	bare := DecodeMessage(tools[0].Value)
	if !bare.Has(FieldMcpDeclName) {
		t.Fatalf("expected the bare tool's name")
	}
	if bare.Has(FieldMcpDeclDesc) {
		t.Fatalf("an absent description must be omitted")
	}
	if bare.Has(FieldMcpDeclSchema) {
		t.Fatalf("an absent parameter schema must be omitted")
	}
	if !bare.Has(FieldMcpDeclServer) {
		t.Fatalf("the server name is always written")
	}

	claude := DecodeMessage(tools[1].Value)
	if !claude.Has(FieldMcpDeclSchema) {
		t.Fatalf("input_schema must be used when parameters is absent")
	}
	if got := string(claude.Get(FieldMcpDeclSchema)[0].Value); !strings.Contains(got, "object") {
		t.Fatalf("unexpected schema: %s", got)
	}
}
