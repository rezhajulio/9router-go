package cursor

// EncodeMcpToolDefinition encodes agent.v1.McpToolDefinition body.
func EncodeMcpToolDefinition(name, description string, schema any) []byte {
	return ConcatBuffers(
		EncodeField(1, WireBytes, name),
		EncodeField(2, WireBytes, description),
		EncodeField(3, WireBytes, EncodeAgentValue(schema)),
		EncodeField(4, WireBytes, "9router"),
		EncodeField(5, WireBytes, name),
	)
}

// EncodeMcpTools encodes multiple tool definitions for AgentRunRequest.mcp_tools.
func EncodeMcpTools(tools []any) []byte {
	if len(tools) == 0 {
		return nil
	}
	var defs [][]byte
	for _, t := range tools {
		tMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		name := ""
		description := ""
		var schema any

		if fn, hasFn := tMap["function"].(map[string]any); hasFn {
			if n, ok := fn["name"].(string); ok {
				name = n
			}
			if d, ok := fn["description"].(string); ok {
				description = d
			}
			schema = fn["parameters"]
		} else {
			if n, ok := tMap["name"].(string); ok {
				name = n
			}
			if d, ok := tMap["description"].(string); ok {
				description = d
			}
			if s, hasS := tMap["parameters"]; hasS {
				schema = s
			} else if s, hasIn := tMap["inputSchema"]; hasIn {
				schema = s
			} else if s, hasIn2 := tMap["input_schema"]; hasIn2 {
				schema = s
			}
		}
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		def := EncodeMcpToolDefinition(name, description, schema)
		defs = append(defs, EncodeField(1, WireBytes, def))
	}
	return ConcatBuffers(defs...)
}

// McpArgs holds decoded MCP tool invocation parameters.
type McpArgs struct {
	Name       string
	ToolName   string
	ToolCallID string
	Args       map[string]any
	// Truncated reports that the payload did not decode cleanly. The arguments
	// would then be a partial view of what Cursor sent, so callers must decline
	// the call instead of handing a half-populated argument object to a client
	// tool whose schema may reject it or, worse, act on incomplete input.
	Truncated bool
}

// DecodeMcpArgs parses an incoming MCP execution request.
func DecodeMcpArgs(bytes []byte) McpArgs {
	msg, err := DecodeMessageStrict(bytes)
	args := make(map[string]any)

	for _, entry := range msg.Get(2) { // 2 = repeated args entry
		pair, pairErr := DecodeMessageStrict(entry.Value)
		if pairErr != nil {
			err = pairErr
			break
		}
		if pair.Has(1) {
			key := string(pair.Get(1)[0].Value)
			if pair.Has(2) {
				args[key] = DecodeAgentValue(pair.Get(2)[0].Value)
			}
		}
	}

	res := McpArgs{Args: args, Truncated: err != nil}
	if msg.Has(1) {
		res.Name = string(msg.Get(1)[0].Value)
	}
	if msg.Has(3) {
		res.ToolCallID = string(msg.Get(3)[0].Value)
	}
	if msg.Has(5) {
		res.ToolName = string(msg.Get(5)[0].Value)
	}
	return res
}

// EncodeMcpResultSuccess encodes success payload for an MCP call.
func EncodeMcpResultSuccess(textItems []string, isError bool) []byte {
	var items [][]byte
	for _, text := range textItems {
		textContent := EncodeField(1, WireBytes, EncodeField(1, WireBytes, text))
		items = append(items, EncodeField(1, WireBytes, textContent))
	}
	errVal := 0
	if isError {
		errVal = 1
	}
	items = append(items, EncodeField(2, WireVarint, errVal))
	success := ConcatBuffers(items...)
	return EncodeField(1, WireBytes, success)
}

// EncodeMcpResultError encodes error payload for an MCP call.
func EncodeMcpResultError(message string) []byte {
	errPayload := EncodeField(1, WireBytes, message)
	return EncodeField(2, WireBytes, errPayload)
}

// EncodeMcpResultToolNotFound encodes tool not found payload.
func EncodeMcpResultToolNotFound(name string) []byte {
	tnfPayload := EncodeField(1, WireBytes, name)
	return EncodeField(5, WireBytes, tnfPayload)
}
