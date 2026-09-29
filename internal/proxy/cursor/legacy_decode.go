package cursor

import (
	"encoding/json"
	"strings"
)

// ExtractLegacyResponse parses a StreamUnifiedChatResponse frame payload with correct field IDs.
func ExtractLegacyResponse(payload []byte) LegacyResponseFrame {
	if res, ok := legacyJSONError(payload); ok {
		return res
	}

	fields := DecodeMessage(payload)
	if tc := legacyToolCallFromFields(fields); tc != nil {
		return LegacyResponseFrame{ToolCall: tc}
	}
	return legacyTextFromFields(fields)
}

// legacyJSONError recognises a Connect end-stream trailer that carries a JSON
// error object instead of a protobuf message, and maps it the way upstream's
// createErrorResponse does: the debug title wins over the debug detail, which
// wins over the plain message, and the code travels separately so the caller can
// decide 429 vs 400.
func legacyJSONError(payload []byte) (LegacyResponseFrame, bool) {
	if len(payload) <= 10 || payload[0] != '{' || !strings.Contains(string(payload), "\"error\"") {
		return LegacyResponseFrame{}, false
	}

	var errMap map[string]any
	if err := json.Unmarshal(payload, &errMap); err != nil {
		return LegacyResponseFrame{Error: string(payload)}, true
	}
	e, ok := errMap["error"].(map[string]any)
	if !ok {
		return LegacyResponseFrame{Error: string(payload)}, true
	}

	res := LegacyResponseFrame{}
	if code, ok := e["code"].(string); ok {
		res.ErrorCode = code
	}
	if msg, ok := legacyErrorTitle(e); ok {
		res.Error = msg
		return res, true
	}
	if msg, ok := e["message"].(string); ok && msg != "" {
		res.Error = msg
		return res, true
	}
	res.Error = string(payload)
	return res, true
}

// legacyErrorTitle reads error.details[0].debug.details.title/detail, the two
// fields upstream prefers over error.message.
func legacyErrorTitle(e map[string]any) (string, bool) {
	details, ok := e["details"].([]any)
	if !ok || len(details) == 0 {
		return "", false
	}
	dMap, ok := details[0].(map[string]any)
	if !ok {
		return "", false
	}
	debug, ok := dMap["debug"].(map[string]any)
	if !ok {
		return "", false
	}
	dbgDetails, ok := debug["details"].(map[string]any)
	if !ok {
		return "", false
	}
	if title, ok := dbgDetails["title"].(string); ok && title != "" {
		return title, true
	}
	if detail, ok := dbgDetails["detail"].(string); ok && detail != "" {
		return detail, true
	}
	return "", false
}

// legacyToolCallFromFields decodes the ClientSideToolV2Call branch (field 1).
// Returns nil when the frame carries no usable tool call.
func legacyToolCallFromFields(fields DecodedMessage) *LegacyToolCall {
	if !fields.Has(FieldLegacyToolCall) {
		return nil
	}
	tcMsg := DecodeMessage(fields.Get(FieldLegacyToolCall)[0].Value)

	id := ""
	if tcMsg.Has(FieldLegacyToolID) {
		// The id field packs the call id and the model call id: "call\nmc_model".
		id = strings.Split(string(tcMsg.Get(FieldLegacyToolID)[0].Value), "\n")[0]
	}
	name := ""
	if tcMsg.Has(FieldLegacyToolName) {
		name = string(tcMsg.Get(FieldLegacyToolName)[0].Value)
	}
	isLast := false
	if tcMsg.Has(FieldLegacyToolIsLast) {
		isLast = tcMsg.Get(FieldLegacyToolIsLast)[0].Varint != 0
	} else if tcMsg.Has(FieldLegacyToolIsLastAlt) {
		isLast = tcMsg.Get(FieldLegacyToolIsLastAlt)[0].Varint != 0
	}

	// The MCP params branch carries the real tool name and arguments; the
	// top-level name exists only for non-MCP calls.
	rawArgs := ""
	if tcMsg.Has(FieldLegacyToolMCPParams) {
		mcpMsg := DecodeMessage(tcMsg.Get(FieldLegacyToolMCPParams)[0].Value)
		if mcpMsg.Has(FieldLegacyMCPToolsList) {
			tMsg := DecodeMessage(mcpMsg.Get(FieldLegacyMCPToolsList)[0].Value)
			if tMsg.Has(FieldLegacyMCPNestedName) {
				name = string(tMsg.Get(FieldLegacyMCPNestedName)[0].Value)
			}
			if tMsg.Has(FieldLegacyMCPNestedParams) {
				rawArgs = string(tMsg.Get(FieldLegacyMCPNestedParams)[0].Value)
			}
		}
	}
	if rawArgs == "" && tcMsg.Has(FieldLegacyToolRawArgs) {
		rawArgs = string(tcMsg.Get(FieldLegacyToolRawArgs)[0].Value)
	}

	if id == "" || name == "" {
		return nil
	}
	if rawArgs == "" {
		rawArgs = "{}"
	}
	return &LegacyToolCall{ID: id, Name: name, Arguments: rawArgs, IsLast: isLast}
}

// legacyTextFromFields decodes the StreamUnifiedChatResponse branch (field 2).
func legacyTextFromFields(fields DecodedMessage) LegacyResponseFrame {
	var res LegacyResponseFrame
	if !fields.Has(FieldLegacyResponse) {
		return res
	}
	respMsg := DecodeMessage(fields.Get(FieldLegacyResponse)[0].Value)
	if respMsg.Has(FieldLegacyResponseText) {
		res.Text = string(respMsg.Get(FieldLegacyResponseText)[0].Value)
	}
	if respMsg.Has(FieldLegacyThinking) {
		thMsg := DecodeMessage(respMsg.Get(FieldLegacyThinking)[0].Value)
		if thMsg.Has(FieldLegacyThinkingText) {
			res.Thinking = string(thMsg.Get(FieldLegacyThinkingText)[0].Value)
		}
	}
	return res
}
