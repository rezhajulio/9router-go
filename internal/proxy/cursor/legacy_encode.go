package cursor

import (
	"encoding/json"

	"github.com/google/uuid"
)

// GenerateLegacyCursorBody builds the Connect-RPC frame with complete upstream fields.
func GenerateLegacyCursorBody(messages []any, modelName string, tools []any, reasoningEffort string, forceAgentMode bool) []byte {
	hasTools := len(tools) > 0
	isAgentic := hasTools || forceAgentMode

	unifiedMode, unifiedModeName := 1, "Ask" // CHAT
	if isAgentic {
		unifiedMode, unifiedModeName = 2, "Agent"
	}

	convID := uuid.New().String()
	callMeta := legacyToolCallMeta(messages)
	encodedMsgs, msgIDParts := legacyEncodeMessages(messages, hasTools, callMeta)

	reqParts := make([][]byte, 0, len(encodedMsgs)+16)
	reqParts = append(reqParts, encodedMsgs...)
	reqParts = append(reqParts,
		EncodeField(FieldLegacyUnknown2, WireVarint, 1),
		EncodeField(FieldLegacyInstruction, WireBytes, []byte{}),
		EncodeField(FieldLegacyUnknown4, WireVarint, 1),
		EncodeLegacyModel(modelName),
		EncodeField(FieldLegacyWebTool, WireBytes, []byte{}),
		EncodeField(FieldLegacyUnknown13, WireVarint, 1),
		EncodeLegacyCursorSetting(),
		EncodeField(FieldLegacyUnknown19, WireVarint, 1),
		EncodeField(FieldLegacyConversationID, WireBytes, convID),
		EncodeLegacyMetadata(),
		EncodeField(FieldLegacyIsAgentic, WireVarint, boolVarint(isAgentic)),
	)
	if isAgentic {
		reqParts = append(reqParts, EncodeField(FieldLegacySupportedTools, WireBytes, EncodeVarint(1)))
	}
	reqParts = append(reqParts, msgIDParts...)
	reqParts = append(reqParts, legacyMcpToolParts(tools)...)
	reqParts = append(reqParts,
		EncodeField(FieldLegacyLargeContext, WireVarint, 0),
		EncodeField(FieldLegacyUnknown38, WireVarint, 0),
		EncodeField(FieldLegacyUnifiedMode, WireVarint, unifiedMode),
		EncodeField(FieldLegacyUnknown47, WireBytes, []byte{}),
		EncodeField(FieldLegacyShouldDisableTools, WireVarint, boolVarint(!isAgentic)),
		EncodeField(FieldLegacyThinkingLevel, WireVarint, legacyThinkingLevel(reasoningEffort)),
		EncodeField(FieldLegacyUnknown51, WireVarint, 0),
		EncodeField(FieldLegacyUnknown53, WireVarint, 1),
		EncodeField(FieldLegacyUnifiedModeName, WireBytes, unifiedModeName),
	)

	requestEnvelope := EncodeField(FieldLegacyRequest, WireBytes, ConcatBuffers(reqParts...))
	return WrapConnectRPCFrame(requestEnvelope)
}

func boolVarint(v bool) int {
	if v {
		return 1
	}
	return 0
}

// legacyThinkingLevel maps OpenAI's reasoning_effort onto Cursor's thinking
// level. Upstream only recognises the exact strings "medium" and "high"; every
// other value (including "max") stays UNSPECIFIED.
func legacyThinkingLevel(effort string) int {
	switch effort {
	case "medium":
		return 1
	case "high":
		return 2
	default:
		return 0
	}
}

// legacyToolMeta is what a role:"tool" message needs from the assistant call it
// answers: an OpenAI tool message carries only tool_call_id, while Cursor's
// ClientSideToolV2Result also needs the tool name and arguments.
type legacyToolMeta struct{ name, arguments string }

// legacyToolCallMeta indexes every assistant tool_call by id so a later
// role:"tool" message can be encoded as a full ToolResult pair.
func legacyToolCallMeta(messages []any) map[string]legacyToolMeta {
	callMeta := make(map[string]legacyToolMeta)
	for _, m := range messages {
		mMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if roleStr, _ := mMap["role"].(string); roleStr != "assistant" {
			continue
		}
		tcs, ok := mMap["tool_calls"].([]any)
		if !ok {
			continue
		}
		for _, tc := range tcs {
			tcMap, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			id, _ := tcMap["id"].(string)
			if id == "" {
				continue
			}
			meta := legacyToolMeta{}
			if fn, ok := tcMap["function"].(map[string]any); ok {
				meta.name, _ = fn["name"].(string)
				meta.arguments, _ = fn["arguments"].(string)
			}
			callMeta[id] = meta
		}
	}
	return callMeta
}

// legacyEncodeMessages converts the OpenAI message list into Cursor message
// fields plus the parallel message-id fields.
func legacyEncodeMessages(messages []any, hasTools bool, callMeta map[string]legacyToolMeta) (encodedMsgs, msgIDParts [][]byte) {
	for i, m := range messages {
		mMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		roleStr, _ := mMap["role"].(string)
		content := TextFromContent(mMap["content"])
		role := legacyRole(roleStr)
		if roleStr == "system" && content != "" && !hasSystemPrefix(content) {
			content = SystemInstructionPrefix + content
		}

		toolResults := legacyInlineToolResults(mMap)
		if roleStr == "tool" {
			toolResults = append(toolResults, legacyToolResult(mMap, callMeta, content))
		}

		msgID := uuid.New().String()
		isLast := i == len(messages)-1
		encodedMsgs = append(encodedMsgs, EncodeLegacyMessage(content, role, msgID, hasTools, isLast, toolResults))

		idEntry := ConcatBuffers(
			EncodeField(1, WireBytes, msgID),
			EncodeField(3, WireVarint, role),
		)
		msgIDParts = append(msgIDParts, EncodeField(FieldLegacyMessageIDs, WireBytes, idEntry))
	}
	return encodedMsgs, msgIDParts
}

// legacyRole maps an OpenAI role onto Cursor's role enum (USER=1, ASSISTANT=2).
// Every non-assistant role travels as USER, matching upstream's mapping.
func legacyRole(role string) int {
	if role == "assistant" {
		return 2
	}
	return 1
}

func hasSystemPrefix(content string) bool {
	return len(content) >= len(SystemInstructionPrefix) && content[:len(SystemInstructionPrefix)] == SystemInstructionPrefix
}

func legacyInlineToolResults(mMap map[string]any) []map[string]any {
	trs, ok := mMap["tool_results"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(trs))
	for _, tr := range trs {
		if trm, ok := tr.(map[string]any); ok {
			out = append(out, trm)
		}
	}
	return out
}

// legacyToolResult builds the ToolResult entry for an OpenAI role:"tool" message.
//
// Deliberate divergence from upstream: the result stays on the USER-role message
// that carries it, whereas upstream's encodeRequest() maps every non-"user" role
// to ASSISTANT and attaches tool_results to the assistant bubble that made the
// call. If Cursor ever validates result ownership against assistant bubbles, the
// carrier role here is the first thing to change.
func legacyToolResult(mMap map[string]any, callMeta map[string]legacyToolMeta, content string) map[string]any {
	callID, _ := mMap["tool_call_id"].(string)
	meta := callMeta[callID]
	name, _ := mMap["name"].(string)
	if name == "" {
		name = meta.name
	}
	args := meta.arguments
	if args == "" {
		args = "{}"
	}
	return map[string]any{
		"tool_call_id":   callID,
		"tool_name":      name,
		"raw_args":       args,
		"result_content": content,
	}
}

// legacyMcpToolParts encodes the client's declared tools as MCP tool fields.
//
// Upstream's encodeMcpTool omits an empty name, an empty description and an
// empty parameter schema, and falls back to `input_schema` for Claude-style
// declarations.
func legacyMcpToolParts(tools []any) [][]byte {
	var parts [][]byte
	for _, t := range tools {
		tMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		name, desc, schema := legacyToolDeclaration(tMap)

		var fields [][]byte
		if name != "" {
			fields = append(fields, EncodeField(FieldMcpDeclName, WireBytes, name))
		}
		if desc != "" {
			fields = append(fields, EncodeField(FieldMcpDeclDesc, WireBytes, desc))
		}
		if schemaMap, ok := schema.(map[string]any); ok && len(schemaMap) > 0 {
			schemaBytes, _ := json.Marshal(schemaMap)
			fields = append(fields, EncodeField(FieldMcpDeclSchema, WireBytes, string(schemaBytes)))
		}
		fields = append(fields, EncodeField(FieldMcpDeclServer, WireBytes, "custom"))
		parts = append(parts, EncodeField(FieldLegacyMcpTools, WireBytes, ConcatBuffers(fields...)))
	}
	return parts
}

// legacyToolDeclaration reads name/description/schema from either OpenAI
// (`{"function":{...}}`) or Claude (`{"name":..., "input_schema":...}`) shape.
func legacyToolDeclaration(tMap map[string]any) (name, desc string, schema any) {
	src := tMap
	if fn, ok := tMap["function"].(map[string]any); ok {
		src = fn
	}
	name, _ = src["name"].(string)
	desc, _ = src["description"].(string)
	if s, ok := src["parameters"]; ok && s != nil {
		schema = s
	} else if s, ok := src["input_schema"]; ok && s != nil {
		schema = s
	}
	return name, desc, schema
}
