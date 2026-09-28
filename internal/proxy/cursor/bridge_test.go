package cursor

import "testing"

func shellExecRequest(variant int, command, workingDir string, timeout, hardTimeout int64, isBackground bool) DecodedMessage {
	args := ConcatBuffers(
		EncodeField(1, WireBytes, command),
		EncodeField(2, WireBytes, workingDir),
	)
	if timeout > 0 {
		args = ConcatBuffers(args, EncodeField(3, WireVarint, uint64(timeout)))
	}
	if hardTimeout > 0 {
		args = ConcatBuffers(args, EncodeField(14, WireVarint, uint64(hardTimeout)))
	}
	if isBackground {
		args = ConcatBuffers(args, EncodeField(11, WireVarint, 1))
	}
	return DecodeMessage(ConcatBuffers(
		EncodeField(1, WireVarint, 7),
		EncodeField(15, WireBytes, "exec-1"),
		EncodeField(variant, WireBytes, args),
	))
}

func readExecRequest(path string) DecodedMessage {
	return DecodeMessage(ConcatBuffers(
		EncodeField(1, WireVarint, 7),
		EncodeField(15, WireBytes, "exec-1"),
		EncodeField(7, WireBytes, EncodeField(1, WireBytes, path)),
	))
}

func bashTool(name string, params map[string]any) ClientTool {
	return ClientTool{Name: name, Parameters: params}
}

func shellSchema(extraProps map[string]any) map[string]any {
	props := map[string]any{
		"command":     map[string]any{"type": "string", "description": "command to run"},
		"description": map[string]any{"type": "string"},
	}
	for k, v := range extraProps {
		props[k] = v
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   []any{"command"},
	}
}

func TestDecodeBuiltinEvent(t *testing.T) {
	tests := []struct {
		name    string
		req     DecodedMessage
		wantNil bool
		kind    string
		command string
		path    string
		bg      bool
		timeout int64
	}{
		{
			name:    "shell decodes command and working dir",
			req:     shellExecRequest(2, "ls -la", "/tmp", 0, 0, false),
			kind:    BuiltinShell,
			command: "ls -la",
		},
		{
			name:    "shell-stream decodes as shell",
			req:     shellExecRequest(14, "echo hi", "", 0, 0, false),
			kind:    BuiltinShell,
			command: "echo hi",
		},
		{
			name:    "background shell keeps flags",
			req:     shellExecRequest(16, "sleep 60", "", 5, 0, true),
			kind:    BuiltinShell,
			command: "sleep 60",
			bg:      true,
			timeout: 5,
		},
		{
			name: "read decodes path",
			req:  readExecRequest("/etc/hosts"),
			kind: BuiltinRead,
			path: "/etc/hosts",
		},
		{
			name: "mcp variant is not a builtin",
			req: DecodeMessage(ConcatBuffers(
				EncodeField(1, WireVarint, 7),
				EncodeField(11, WireBytes, EncodeField(5, WireBytes, "mytool")),
			)),
			wantNil: true,
		},
		{
			name: "request context is not a builtin",
			req: DecodeMessage(ConcatBuffers(
				EncodeField(1, WireVarint, 7),
				EncodeField(10, WireBytes, []byte{}),
			)),
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := DecodeBuiltinEvent(tt.req)
			if tt.wantNil {
				if ev != nil {
					t.Fatalf("expected nil, got %+v", ev)
				}
				return
			}
			if ev == nil {
				t.Fatalf("expected event, got nil")
			}
			if ev.Kind != tt.kind {
				t.Fatalf("kind = %q, want %q", ev.Kind, tt.kind)
			}
			if ev.Command != tt.command {
				t.Fatalf("command = %q, want %q", ev.Command, tt.command)
			}
			if ev.Path != tt.path {
				t.Fatalf("path = %q, want %q", ev.Path, tt.path)
			}
			if ev.IsBackground != tt.bg {
				t.Fatalf("isBackground = %v, want %v", ev.IsBackground, tt.bg)
			}
			if ev.Timeout != tt.timeout {
				t.Fatalf("timeout = %d, want %d", ev.Timeout, tt.timeout)
			}
		})
	}
}

func TestBridgeShell(t *testing.T) {
	tools := []ClientTool{bashTool("bash", shellSchema(nil))}

	t.Run("bridges onto bash with required command", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls -la", "/tmp", 0, 0, false))
		b := BridgeBuiltinTool(ev, tools)
		if b == nil {
			t.Fatalf("expected bridge, got nil")
		}
		if b.ToolName != "bash" {
			t.Fatalf("tool = %q, want bash", b.ToolName)
		}
		if b.Arguments["command"] != "ls -la" {
			t.Fatalf("args = %v", b.Arguments)
		}
	})

	t.Run("timeout fails closed", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls", "", 30, 0, false))
		if b := BridgeBuiltinTool(ev, tools); b != nil {
			t.Fatalf("timeout must fail closed, got %+v", b)
		}
	})

	t.Run("hard timeout fails closed", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls", "", 0, 60, false))
		if b := BridgeBuiltinTool(ev, tools); b != nil {
			t.Fatalf("hard timeout must fail closed, got %+v", b)
		}
	})

	t.Run("empty command fails closed", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(2, "   ", "", 0, 0, false))
		if b := BridgeBuiltinTool(ev, tools); b != nil {
			t.Fatalf("empty command must fail closed, got %+v", b)
		}
	})

	t.Run("background fails closed without platform", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(16, "sleep 60", "", 0, 0, true))
		if b := BridgeBuiltinTool(ev, tools); b != nil {
			t.Fatalf("background shell must fail closed, got %+v", b)
		}
	})

	t.Run("no matching tool fails closed", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls", "", 0, 0, false))
		if b := BridgeBuiltinTool(ev, []ClientTool{bashTool("read", shellSchema(nil))}); b != nil {
			t.Fatalf("expected nil without a shell tool, got %+v", b)
		}
	})

	t.Run("unsupported schema keyword fails closed", func(t *testing.T) {
		bad := shellSchema(map[string]any{
			"command": map[string]any{"type": "string", "pattern": "^ls"},
		})
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls", "", 0, 0, false))
		if b := BridgeBuiltinTool(ev, []ClientTool{bashTool("bash", bad)}); b != nil {
			t.Fatalf("unsupported schema keyword must fail closed, got %+v", b)
		}
	})

	t.Run("missing required arg fails closed", func(t *testing.T) {
		schema := shellSchema(map[string]any{
			"session": map[string]any{"type": "string"},
		})
		schema["required"] = []any{"command", "session"}
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls", "", 0, 0, false))
		if b := BridgeBuiltinTool(ev, []ClientTool{bashTool("bash", schema)}); b != nil {
			t.Fatalf("unfillable required key must fail closed, got %+v", b)
		}
	})

	t.Run("name match is case-insensitive", func(t *testing.T) {
		ev := DecodeBuiltinEvent(shellExecRequest(2, "ls", "", 0, 0, false))
		if b := BridgeBuiltinTool(ev, []ClientTool{bashTool("Bash", shellSchema(nil))}); b == nil {
			t.Fatalf("expected case-insensitive match")
		}
	})
}

func TestBridgeRead(t *testing.T) {
	readSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string"},
		},
		"required": []any{"path"},
	}
	tools := []ClientTool{bashTool("read", readSchema)}

	t.Run("bridges onto read", func(t *testing.T) {
		ev := DecodeBuiltinEvent(readExecRequest("/etc/hosts"))
		b := BridgeBuiltinTool(ev, tools)
		if b == nil {
			t.Fatalf("expected bridge, got nil")
		}
		if b.ToolName != "read" || b.Arguments["path"] != "/etc/hosts" {
			t.Fatalf("unexpected bridge %+v", b)
		}
	})

	t.Run("no matching tool fails closed", func(t *testing.T) {
		ev := DecodeBuiltinEvent(readExecRequest("/etc/hosts"))
		if b := BridgeBuiltinTool(ev, []ClientTool{bashTool("bash", shellSchema(nil))}); b != nil {
			t.Fatalf("expected nil without a read tool, got %+v", b)
		}
	})

	t.Run("nil event fails closed", func(t *testing.T) {
		if b := BridgeBuiltinTool(nil, tools); b != nil {
			t.Fatalf("expected nil event to fail closed, got %+v", b)
		}
	})
}

func TestParseClientTools(t *testing.T) {
	tools := []any{
		map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        "bash",
				"description": "run a command",
				"parameters":  shellSchema(nil),
			},
		},
		map[string]any{"type": "function"}, // no function block: skipped
		"not-a-tool",                       // skipped
	}
	got := ParseClientTools(tools)
	if len(got) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(got))
	}
	if got[0].Name != "bash" {
		t.Fatalf("name = %q", got[0].Name)
	}
	if got[0].Parameters["type"] != "object" {
		t.Fatalf("parameters not parsed: %v", got[0].Parameters)
	}
}

func TestRejectExecRequestShellStream(t *testing.T) {
	// Shell-stream (14) shares the shell result field (2), matching
	// OmniRoute's encodeExecShellRejected (ECM_SHELL_RESULT).
	req := shellExecRequest(14, "echo hi", "", 0, 0, false)
	rej := RejectExecRequest(req)
	if rej == nil {
		t.Fatalf("expected rejection frame for variant 14")
	}
	dec := DecodeMessage(rej[5:])
	if !dec.Has(2) {
		t.Fatalf("expected ExecClientMessage (field 2)")
	}
	execClient := DecodeMessage(dec.Get(2)[0].Value)
	if !execClient.Has(2) {
		t.Fatalf("expected shell result payload under field 2")
	}
}
