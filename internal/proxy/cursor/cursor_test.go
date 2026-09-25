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
