package cursor

import "strings"

// Builtin tool bridging (OmniRoute parity).
//
// Cursor's AgentService frequently asks the client to run its native IDE
// builtins (shell, read, ...) even when the OpenAI caller declared its own
// external tools. This proxy has no IDE to run them in, so instead of only
// declining the request it bridges the call onto a schema-compatible tool
// the caller declared: the native request is surfaced as a normal OpenAI
// tool_call, the typed rejection still goes upstream, and the harness
// executes the call and returns the result on the next turn.
//
// Wire layout is taken from OmniRoute's decoder
// (open-sse/utils/cursorAgentProtobuf.ts): ExecServerMessage oneof
// discriminators ESM_SHELL_ARGS=2, ESM_READ_ARGS=7,
// ESM_SHELL_STREAM_ARGS=14, ESM_BACKGROUND_SHELL_SPAWN=16, with
// ShellArgs{command=1, working_directory=2, timeout=3, is_background=11,
// hard_timeout=14} and ReadArgs{path=1}.

// Builtin event kinds produced by DecodeBuiltinEvent.
const (
	BuiltinShell = "shell"
	BuiltinRead  = "read"
)

// ExecServerMessage oneof discriminators for the bridgeable IDE builtins.
const (
	execVariantShell       = 2
	execVariantRead        = 7
	execVariantShellStream = 14
	execVariantBgShell     = 16
)

// ShellArgs field numbers.
const (
	shellArgCommand      = 1
	shellArgWorkingDir   = 2
	shellArgTimeout      = 3
	shellArgIsBackground = 11
	shellArgHardTimeout  = 14
)

// ReadArgs path field number.
const readArgPath = 1

// BuiltinEvent is a decoded IDE builtin exec request that may be bridged
// onto one of the client's declared tools.
type BuiltinEvent struct {
	Kind         string
	Command      string
	WorkingDir   string
	Timeout      int64
	HardTimeout  int64
	IsBackground bool
	Path         string
}

func stringField(msg DecodedMessage, field int) string {
	if msg.Has(field) {
		if f := msg.Get(field); len(f) > 0 && f[0].WireType == WireBytes {
			return string(f[0].Value)
		}
	}
	return ""
}

func varintField(msg DecodedMessage, field int) int64 {
	if msg.Has(field) {
		if f := msg.Get(field); len(f) > 0 && f[0].WireType == WireVarint {
			return int64(f[0].Varint)
		}
	}
	return 0
}

// DecodeBuiltinEvent decodes a shell / shell-stream / background-shell /
// read exec variant from an ExecServerMessage. It returns nil for anything
// else, including request_context (10) and MCP (11).
func DecodeBuiltinEvent(execRequest DecodedMessage) *BuiltinEvent {
	variant := ExecRequestVariant(execRequest)
	switch variant {
	case execVariantShell, execVariantShellStream, execVariantBgShell:
		args := DecodeMessage(execRequest.Get(variant)[0].Value)
		return &BuiltinEvent{
			Kind:         BuiltinShell,
			Command:      stringField(args, shellArgCommand),
			WorkingDir:   stringField(args, shellArgWorkingDir),
			Timeout:      varintField(args, shellArgTimeout),
			HardTimeout:  varintField(args, shellArgHardTimeout),
			IsBackground: variant == execVariantBgShell || varintField(args, shellArgIsBackground) != 0,
		}
	case execVariantRead:
		args := DecodeMessage(execRequest.Get(variant)[0].Value)
		return &BuiltinEvent{
			Kind: BuiltinRead,
			Path: stringField(args, readArgPath),
		}
	}
	return nil
}

// ClientTool is an OpenAI function tool definition declared by the caller.
type ClientTool struct {
	Name       string
	Parameters map[string]any
}

// ParseClientTools extracts name + JSON schema from OpenAI-format tool
// definitions ({"type":"function","function":{"name":...,"parameters":...}}).
func ParseClientTools(tools []any) []ClientTool {
	var out []ClientTool
	for _, t := range tools {
		m, ok := t.(map[string]any)
		if !ok {
			continue
		}
		fn, ok := m["function"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := fn["name"].(string)
		if name == "" {
			continue
		}
		params, _ := fn["parameters"].(map[string]any)
		out = append(out, ClientTool{Name: name, Parameters: params})
	}
	return out
}

// FilterClientToolsByChoice restricts client tools according to OpenAI tool_choice.
func FilterClientToolsByChoice(tools []ClientTool, toolChoice any) []ClientTool {
	if toolChoice == nil {
		return tools
	}
	if s, ok := toolChoice.(string); ok {
		switch s {
		case "none":
			return nil
		case "auto", "required", "":
			return tools
		default:
			return nil
		}
	}
	if tcMap, ok := toolChoice.(map[string]any); ok {
		if tcMap["type"] == "function" {
			if fn, ok := tcMap["function"].(map[string]any); ok {
				if name, ok := fn["name"].(string); ok && name != "" {
					var filtered []ClientTool
					for _, t := range tools {
						if t.Name == name {
							filtered = append(filtered, t)
						}
					}
					return filtered
				}
			}
		}
	}
	return nil
}

// BuiltinBridge is a native builtin request rewritten as a call to one of
// the client's own tools.
type BuiltinBridge struct {
	ToolName  string
	Arguments map[string]any
}

func namedTools(tools []ClientTool, names []string) []ClientTool {
	var out []ClientTool
	for _, n := range names {
		for _, t := range tools {
			if strings.EqualFold(t.Name, n) {
				out = append(out, t)
			}
		}
	}
	return out
}

// Schema validation below is fail-closed: only the small schema subset for
// which generated values are proven valid is accepted. Any validation
// keyword the bridge does not implement fails the bridge, keeping the
// native typed rejection.

var rootSchemaKeys = map[string]bool{
	"$schema": true, "$id": true, "$comment": true, "title": true,
	"description": true, "type": true, "properties": true,
	"required": true, "additionalProperties": true,
}

var scalarPropertyKeys = map[string]bool{
	"type": true, "$comment": true, "title": true, "description": true,
	"default": true, "examples": true, "deprecated": true,
	"readOnly": true, "writeOnly": true,
}

func objectSchema(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || m["type"] != "object" {
		return nil
	}
	for k := range m {
		if !rootSchemaKeys[k] {
			return nil
		}
	}
	if p, ok := m["properties"]; ok && p != nil {
		if _, ok := p.(map[string]any); !ok {
			return nil
		}
	}
	if r, ok := m["required"]; ok && r != nil {
		arr, ok := r.([]any)
		if !ok {
			return nil
		}
		for _, e := range arr {
			if _, ok := e.(string); !ok {
				return nil
			}
		}
	}
	if ap, ok := m["additionalProperties"]; ok && ap != nil {
		if _, ok := ap.(bool); !ok {
			return nil
		}
	}
	return m
}

func stringProperty(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	for k := range m {
		if !scalarPropertyKeys[k] {
			return false
		}
	}
	return m["type"] == "string"
}

func schemaProperties(schema map[string]any) map[string]any {
	if p, ok := schema["properties"].(map[string]any); ok {
		return p
	}
	return nil
}

func requiredKeys(schema map[string]any) map[string]bool {
	out := map[string]bool{}
	if r, ok := schema["required"].([]any); ok {
		for _, e := range r {
			if s, ok := e.(string); ok {
				out[s] = true
			}
		}
	}
	return out
}

func selectStringProperty(schema map[string]any, props map[string]any, names []string) string {
	required := requiredKeys(schema)
	for _, name := range names {
		if required[name] && stringProperty(props[name]) {
			return name
		}
	}
	for _, name := range names {
		if stringProperty(props[name]) {
			return name
		}
	}
	return ""
}

func hasAllRequired(schema map[string]any, args map[string]any) bool {
	for name := range requiredKeys(schema) {
		if _, ok := args[name]; !ok {
			return false
		}
	}
	return true
}

var shellToolNames = []string{"bash", "shell", "run_terminal_cmd"}
var readToolNames = []string{"read", "read_file"}

// bridgeShell ports OmniRoute's directShellBridge: a native shell request
// becomes a call to the client's bash/shell/run_terminal_cmd tool.
func bridgeShell(event *BuiltinEvent, tools []ClientTool) *BuiltinBridge {
	if strings.TrimSpace(event.Command) == "" {
		return nil
	}
	// The external schemas supported here do not expose Cursor's timeout or
	// hard-timeout semantics. Dropping either limit could broaden execution,
	// so preserve the native typed rejection instead of emitting an unsafe
	// call. The check is != 0 rather than > 0 because a negative proto int32
	// sign-extends to a varint above 2^63, which varintField casts back to a
	// negative int64: a set timeout would otherwise read as "unset" and the
	// limit would be silently dropped.
	if event.Timeout != 0 || event.HardTimeout != 0 {
		return nil
	}
	// Background shells bridge onto pty_spawn, which needs the client
	// platform this proxy does not know; fail closed.
	if event.IsBackground {
		return nil
	}
	for _, tool := range namedTools(tools, shellToolNames) {
		schema := objectSchema(tool.Parameters)
		if schema == nil {
			continue
		}
		props := schemaProperties(schema)
		commandKey := selectStringProperty(schema, props, []string{"command", "cmd"})
		if commandKey == "" {
			continue
		}
		args := map[string]any{commandKey: event.Command}
		if cwdKey := selectStringProperty(schema, props, []string{"workdir", "cwd", "workingDirectory", "working_directory"}); cwdKey != "" && event.WorkingDir != "" {
			args[cwdKey] = event.WorkingDir
		}
		if stringProperty(props["description"]) {
			args["description"] = "Run Cursor-requested shell command"
		}
		if hasAllRequired(schema, args) {
			return &BuiltinBridge{ToolName: tool.Name, Arguments: args}
		}
	}
	return nil
}

// bridgeRead ports OmniRoute's readBridge: a native read becomes a call to
// the client's read/read_file tool.
func bridgeRead(event *BuiltinEvent, tools []ClientTool) *BuiltinBridge {
	for _, tool := range namedTools(tools, readToolNames) {
		schema := objectSchema(tool.Parameters)
		if schema == nil {
			continue
		}
		props := schemaProperties(schema)
		pathKey := selectStringProperty(schema, props, []string{"filePath", "path", "file_path"})
		if pathKey == "" {
			continue
		}
		args := map[string]any{pathKey: event.Path}
		if hasAllRequired(schema, args) {
			return &BuiltinBridge{ToolName: tool.Name, Arguments: args}
		}
	}
	return nil
}

// BridgeBuiltinTool converts a Cursor-native builtin request into a declared
// client tool call. Only event variants whose complete arguments are decoded
// are supported. Unknown or constrained schemas fail closed and the caller
// keeps the native typed rejection.
func BridgeBuiltinTool(event *BuiltinEvent, tools []ClientTool) *BuiltinBridge {
	if event == nil {
		return nil
	}
	switch event.Kind {
	case BuiltinRead:
		return bridgeRead(event, tools)
	case BuiltinShell:
		return bridgeShell(event, tools)
	}
	return nil
}
