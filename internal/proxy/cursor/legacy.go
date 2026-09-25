package cursor

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Legacy schema field constants (matching upstream open-sse/utils/cursorProtobuf.js)
const (
	FieldLegacyRequest            = 1
	FieldLegacyMessages           = 1
	FieldLegacyUnknown2           = 2
	FieldLegacyInstruction        = 3
	FieldLegacyUnknown4           = 4
	FieldLegacyModel              = 5
	FieldLegacyWebTool            = 8
	FieldLegacyUnknown13          = 13
	FieldLegacyCursorSetting      = 15
	FieldLegacyUnknown19          = 19
	FieldLegacyConversationID     = 23
	FieldLegacyMetadata           = 26
	FieldLegacyIsAgentic          = 27
	FieldLegacySupportedTools     = 29
	FieldLegacyMessageIDs         = 30
	FieldLegacyMcpTools           = 34
	FieldLegacyLargeContext       = 35
	FieldLegacyUnknown38          = 38
	FieldLegacyUnifiedMode        = 46
	FieldLegacyUnknown47          = 47
	FieldLegacyShouldDisableTools = 48
	FieldLegacyThinkingLevel      = 49
	FieldLegacyUnknown51          = 51
	FieldLegacyUnknown53          = 53
	FieldLegacyUnifiedModeName    = 54

	// Message fields
	FieldLegacyMsgContent        = 1
	FieldLegacyMsgRole           = 2
	FieldLegacyMsgID             = 13
	FieldLegacyMsgToolResults    = 18
	FieldLegacyMsgIsAgentic      = 29
	FieldLegacyMsgServerBubbleID = 32
	FieldLegacyMsgUnifiedMode    = 47
	FieldLegacyMsgSupportedTools = 51

	// Response fields
	FieldLegacyToolCall         = 1
	FieldLegacyResponse         = 2
	FieldLegacyToolID           = 3
	FieldLegacyToolName         = 9
	FieldLegacyToolRawArgs      = 10
	FieldLegacyToolIsLast       = 11
	FieldLegacyToolIsLastAlt    = 15
	FieldLegacyToolMCPParams    = 27
	FieldLegacyMCPToolsList     = 1
	FieldLegacyMCPNestedName    = 1
	FieldLegacyMCPNestedParams  = 3
	FieldLegacyResponseText     = 1
	FieldLegacyThinking         = 25
	FieldLegacyThinkingText     = 1

	// ToolResult structure fields
	FieldToolResultCallID      = 1
	FieldToolResultName        = 2
	FieldToolResultIndex       = 3
	FieldToolResultRawArgs     = 5
	FieldToolResultResult      = 8
	FieldToolResultToolCall    = 11
	FieldToolResultModelCallID = 12

	// ClientSideToolV2Result
	FieldCV2RTool        = 1
	FieldCV2RMcpResult   = 28
	FieldCV2RCallID      = 35
	FieldCV2RModelCallID = 48
	FieldCV2RToolIndex   = 49

	// MCPResult
	FieldMCPRSelectedTool = 1
	FieldMCPRResult       = 2

	// ClientSideToolV2Call
	FieldCV2CTool        = 1
	FieldCV2CMcpParams   = 27
	FieldCV2CCallID      = 3
	FieldCV2CName        = 9
	FieldCV2CRawArgs     = 10
	FieldCV2CToolIndex   = 48
	FieldCV2CModelCallID = 49

	// MCP Params nested
	FieldMCPToolName   = 1
	FieldMCPToolParams = 2
	FieldMCPToolServer = 3
)

// LegacyToolCall holds tool call extracted from ChatService response.
type LegacyToolCall struct {
	ID        string
	Name      string
	Arguments string
	IsLast    bool
}

// LegacyResponseFrame holds parsed content from ChatService response frame.
type LegacyResponseFrame struct {
	Text     string
	Thinking string
	ToolCall *LegacyToolCall
	Error    string
}

func parseToolID(id string) (toolCallID string, modelCallID string) {
	delimiter := "\nmc_"
	idx := strings.Index(id, delimiter)
	if idx >= 0 {
		return id[:idx], id[idx+len(delimiter):]
	}
	return id, ""
}

func formatToolName(name string) string {
	base := name
	if base == "" {
		base = "tool"
	}
	if strings.HasPrefix(base, "mcp__") {
		rest := base[len("mcp__"):]
		splitIdx := strings.Index(rest, "__")
		if splitIdx >= 0 {
			server := rest[:splitIdx]
			toolName := rest[splitIdx+2:]
			if server == "" {
				server = "custom"
			}
			return fmt.Sprintf("mcp_%s_%s", server, toolName)
		}
		return fmt.Sprintf("mcp_custom_%s", rest)
	}
	if strings.HasPrefix(base, "mcp_") {
		return base
	}
	return fmt.Sprintf("mcp_custom_%s", base)
}

func parseToolName(formattedName string) (serverName, selectedTool string) {
	if !strings.HasPrefix(formattedName, "mcp_") {
		return "custom", formattedName
	}
	tail := formattedName[len("mcp_"):]
	splitIdx := strings.Index(tail, "_")
	if splitIdx < 0 {
		return "custom", tail
	}
	return tail[:splitIdx], tail[splitIdx+1:]
}

func encodeMcpResult(selectedTool, resultContent string) []byte {
	return ConcatBuffers(
		EncodeField(FieldMCPRSelectedTool, WireBytes, selectedTool),
		EncodeField(FieldMCPRResult, WireBytes, resultContent),
	)
}

func encodeClientSideToolV2Result(toolCallID, modelCallID, selectedTool, resultContent string, toolIndex int) []byte {
	if toolIndex <= 0 {
		toolIndex = 1
	}
	var parts [][]byte
	parts = append(parts, EncodeField(FieldCV2RTool, WireVarint, 19)) // MCP
	parts = append(parts, EncodeField(FieldCV2RMcpResult, WireBytes, encodeMcpResult(selectedTool, resultContent)))
	parts = append(parts, EncodeField(FieldCV2RCallID, WireBytes, toolCallID))
	if modelCallID != "" {
		parts = append(parts, EncodeField(FieldCV2RModelCallID, WireBytes, modelCallID))
	}
	parts = append(parts, EncodeField(FieldCV2RToolIndex, WireVarint, toolIndex))
	return ConcatBuffers(parts...)
}

func encodeMcpParamsForCall(toolName, rawArgs, serverName string) []byte {
	tool := ConcatBuffers(
		EncodeField(FieldMCPToolName, WireBytes, toolName),
		EncodeField(FieldMCPToolParams, WireBytes, rawArgs),
		EncodeField(FieldMCPToolServer, WireBytes, serverName),
	)
	return EncodeField(FieldLegacyMCPToolsList, WireBytes, tool)
}

func encodeClientSideToolV2Call(toolCallID, toolName, selectedTool, serverName, rawArgs, modelCallID string, toolIndex int) []byte {
	if toolIndex <= 0 {
		toolIndex = 1
	}
	var parts [][]byte
	parts = append(parts, EncodeField(FieldCV2CTool, WireVarint, 19))
	parts = append(parts, EncodeField(FieldCV2CMcpParams, WireBytes, encodeMcpParamsForCall(selectedTool, rawArgs, serverName)))
	parts = append(parts, EncodeField(FieldCV2CCallID, WireBytes, toolCallID))
	parts = append(parts, EncodeField(FieldCV2CName, WireBytes, toolName))
	parts = append(parts, EncodeField(FieldCV2CRawArgs, WireBytes, rawArgs))
	parts = append(parts, EncodeField(FieldCV2CToolIndex, WireVarint, toolIndex))
	if modelCallID != "" {
		parts = append(parts, EncodeField(FieldCV2CModelCallID, WireBytes, modelCallID))
	}
	return ConcatBuffers(parts...)
}

// EncodeToolResult encodes tool result with full ClientSideToolV2Result & Call
func EncodeToolResult(tr map[string]any) []byte {
	rawName, _ := tr["tool_name"].(string)
	if rawName == "" {
		rawName, _ = tr["name"].(string)
	}
	toolName := formatToolName(rawName)
	rawArgs, _ := tr["raw_args"].(string)
	if rawArgs == "" {
		rawArgs = "{}"
	}
	resultContent, _ := tr["result_content"].(string)
	if resultContent == "" {
		resultContent, _ = tr["result"].(string)
	}
	callIDStr, _ := tr["tool_call_id"].(string)
	toolCallID, modelCallID := parseToolID(callIDStr)
	serverName, selectedTool := parseToolName(toolName)

	toolIndex := 1

	var parts [][]byte
	parts = append(parts, EncodeField(FieldToolResultCallID, WireBytes, toolCallID))
	parts = append(parts, EncodeField(FieldToolResultName, WireBytes, toolName))
	parts = append(parts, EncodeField(FieldToolResultIndex, WireVarint, toolIndex))
	if modelCallID != "" {
		parts = append(parts, EncodeField(FieldToolResultModelCallID, WireBytes, modelCallID))
	}
	parts = append(parts, EncodeField(FieldToolResultRawArgs, WireBytes, rawArgs))
	parts = append(parts, EncodeField(FieldToolResultResult, WireBytes,
		encodeClientSideToolV2Result(toolCallID, modelCallID, selectedTool, resultContent, toolIndex)))
	parts = append(parts, EncodeField(FieldToolResultToolCall, WireBytes,
		encodeClientSideToolV2Call(toolCallID, toolName, selectedTool, serverName, rawArgs, modelCallID, toolIndex)))

	return ConcatBuffers(parts...)
}

// EncodeLegacyModel encodes Model field (5).
func EncodeLegacyModel(modelName string) []byte {
	return EncodeField(FieldLegacyModel, WireBytes, ConcatBuffers(
		EncodeField(1, WireBytes, modelName),
		EncodeField(4, WireBytes, []byte{}),
	))
}

// EncodeLegacyCursorSetting encodes real CursorSetting field (15).
func EncodeLegacyCursorSetting() []byte {
	unknown6 := ConcatBuffers(
		EncodeField(1, WireBytes, []byte{}),
		EncodeField(2, WireBytes, []byte{}),
	)
	return EncodeField(FieldLegacyCursorSetting, WireBytes, ConcatBuffers(
		EncodeField(1, WireBytes, "cursor\\aisettings"),
		EncodeField(3, WireBytes, []byte{}),
		EncodeField(6, WireBytes, unknown6),
		EncodeField(8, WireVarint, 1),
		EncodeField(9, WireVarint, 1),
	))
}

// EncodeLegacyMetadata encodes real Metadata field (26).
func EncodeLegacyMetadata() []byte {
	plat := runtime.GOOS
	if plat == "" {
		plat = "linux"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return EncodeField(FieldLegacyMetadata, WireBytes, ConcatBuffers(
		EncodeField(1, WireBytes, plat),
		EncodeField(2, WireBytes, arch),
		EncodeField(3, WireBytes, "v20.0.0"),
		EncodeField(4, WireBytes, "/"),
		EncodeField(5, WireBytes, time.Now().UTC().Format(time.RFC3339)),
	))
}

// EncodeLegacyMessage encodes a single conversation message with full flags.
func EncodeLegacyMessage(content string, role int, msgID string, hasTools bool, isLast bool, toolResults []map[string]any) []byte {
	modeVal := 1 // CHAT
	agenticVal := 0
	if hasTools {
		modeVal = 2 // AGENT
		agenticVal = 1
	}

	var parts [][]byte
	parts = append(parts, EncodeField(FieldLegacyMsgContent, WireBytes, content))
	parts = append(parts, EncodeField(FieldLegacyMsgRole, WireVarint, role))
	if msgID != "" {
		parts = append(parts, EncodeField(FieldLegacyMsgID, WireBytes, msgID))
	}
	for _, tr := range toolResults {
		parts = append(parts, EncodeField(FieldLegacyMsgToolResults, WireBytes, EncodeToolResult(tr)))
	}
	parts = append(parts, EncodeField(FieldLegacyMsgIsAgentic, WireVarint, agenticVal))
	parts = append(parts, EncodeField(FieldLegacyMsgUnifiedMode, WireVarint, modeVal))
	if isLast && hasTools {
		parts = append(parts, EncodeField(FieldLegacyMsgSupportedTools, WireBytes, EncodeVarint(1)))
	}
	return EncodeField(FieldLegacyMessages, WireBytes, ConcatBuffers(parts...))
}

// GenerateLegacyCursorBody builds the Connect-RPC frame with complete upstream fields.
func GenerateLegacyCursorBody(messages []any, modelName string, tools []any, reasoningEffort string, forceAgentMode bool) []byte {
	hasTools := len(tools) > 0
	isAgentic := hasTools || forceAgentMode
	unifiedMode := 1 // CHAT
	unifiedModeName := "Ask"
	if isAgentic {
		unifiedMode = 2 // AGENT
		unifiedModeName = "Agent"
	}

	thinkingLevel := 0
	switch strings.ToLower(reasoningEffort) {
	case "medium":
		thinkingLevel = 1
	case "high", "max":
		thinkingLevel = 2
	}

	convID := uuid.New().String()
	var encodedMsgs [][]byte
	var msgIDParts [][]byte

	for i, m := range messages {
		mMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		roleStr, _ := mMap["role"].(string)
		content := TextFromContent(mMap["content"])
		role := 1 // USER
		if roleStr == "assistant" {
			role = 2
		}

		var toolResults []map[string]any
		if trs, ok := mMap["tool_results"].([]any); ok {
			for _, tr := range trs {
				if trm, ok := tr.(map[string]any); ok {
					toolResults = append(toolResults, trm)
				}
			}
		}

		msgID := uuid.New().String()
		isLast := i == len(messages)-1
		encodedMsgs = append(encodedMsgs, EncodeLegacyMessage(content, role, msgID, isAgentic, isLast, toolResults))

		idEntry := ConcatBuffers(
			EncodeField(1, WireBytes, msgID),
			EncodeField(3, WireVarint, role),
		)
		msgIDParts = append(msgIDParts, EncodeField(FieldLegacyMessageIDs, WireBytes, idEntry))
	}

	var reqParts [][]byte
	reqParts = append(reqParts, encodedMsgs...)
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown2, WireVarint, 1))
	reqParts = append(reqParts, EncodeField(FieldLegacyInstruction, WireBytes, []byte{}))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown4, WireVarint, 1))
	reqParts = append(reqParts, EncodeLegacyModel(modelName))
	reqParts = append(reqParts, EncodeField(FieldLegacyWebTool, WireBytes, []byte{}))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown13, WireVarint, 1))
	reqParts = append(reqParts, EncodeLegacyCursorSetting())
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown19, WireVarint, 1))
	reqParts = append(reqParts, EncodeField(FieldLegacyConversationID, WireBytes, convID))
	reqParts = append(reqParts, EncodeLegacyMetadata())
	agenticVal := 0
	if isAgentic {
		agenticVal = 1
	}
	reqParts = append(reqParts, EncodeField(FieldLegacyIsAgentic, WireVarint, agenticVal))
	if isAgentic {
		reqParts = append(reqParts, EncodeField(FieldLegacySupportedTools, WireBytes, EncodeVarint(1)))
	}
	if len(msgIDParts) > 0 {
		reqParts = append(reqParts, msgIDParts...)
	}

	// MCP Tools
	for _, t := range tools {
		tMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		tName := ""
		tDesc := ""
		var tSchema any = map[string]any{}
		if fn, ok := tMap["function"].(map[string]any); ok {
			if n, ok := fn["name"].(string); ok {
				tName = n
			}
			if d, ok := fn["description"].(string); ok {
				tDesc = d
			}
			if s, ok := fn["parameters"]; ok {
				tSchema = s
			}
		} else {
			if n, ok := tMap["name"].(string); ok {
				tName = n
			}
			if d, ok := tMap["description"].(string); ok {
				tDesc = d
			}
			if s, ok := tMap["parameters"]; ok {
				tSchema = s
			}
		}
		schemaBytes, _ := json.Marshal(tSchema)
		mcpPart := ConcatBuffers(
			EncodeField(1, WireBytes, tName),
			EncodeField(2, WireBytes, tDesc),
			EncodeField(3, WireBytes, string(schemaBytes)),
			EncodeField(4, WireBytes, "custom"),
		)
		reqParts = append(reqParts, EncodeField(FieldLegacyMcpTools, WireBytes, mcpPart))
	}

	disableTools := 1
	if isAgentic {
		disableTools = 0
	}

	reqParts = append(reqParts, EncodeField(FieldLegacyLargeContext, WireVarint, 0))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown38, WireVarint, 0))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnifiedMode, WireVarint, unifiedMode))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown47, WireBytes, []byte{}))
	reqParts = append(reqParts, EncodeField(FieldLegacyShouldDisableTools, WireVarint, disableTools))
	reqParts = append(reqParts, EncodeField(FieldLegacyThinkingLevel, WireVarint, thinkingLevel))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown51, WireVarint, 0))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnknown53, WireVarint, 1))
	reqParts = append(reqParts, EncodeField(FieldLegacyUnifiedModeName, WireBytes, unifiedModeName))

	requestEnvelope := EncodeField(FieldLegacyRequest, WireBytes, ConcatBuffers(reqParts...))
	return WrapConnectRPCFrame(requestEnvelope)
}

// ExtractLegacyResponse parses a StreamUnifiedChatResponse frame payload with correct field IDs.
func ExtractLegacyResponse(payload []byte) LegacyResponseFrame {
	var res LegacyResponseFrame
	if len(payload) > 10 && payload[0] == '{' && strings.Contains(string(payload), "\"error\"") {
		var errMap map[string]any
		if err := json.Unmarshal(payload, &errMap); err == nil {
			if e, ok := errMap["error"].(map[string]any); ok {
				if msg, ok := e["message"].(string); ok {
					res.Error = msg
					return res
				}
			}
		}
		res.Error = string(payload)
		return res
	}

	fields := DecodeMessage(payload)

	// Field 1: ClientSideToolV2Call
	if fields.Has(FieldLegacyToolCall) {
		tcMsg := DecodeMessage(fields.Get(FieldLegacyToolCall)[0].Value)
		id := ""
		name := ""
		rawArgs := ""
		isLast := false

		// Field 3: TOOL_ID
		if tcMsg.Has(FieldLegacyToolID) {
			fullID := string(tcMsg.Get(FieldLegacyToolID)[0].Value)
			id = strings.Split(fullID, "\n")[0]
		}
		// Field 9: TOOL_NAME
		if tcMsg.Has(FieldLegacyToolName) {
			name = string(tcMsg.Get(FieldLegacyToolName)[0].Value)
		}
		// Field 11: TOOL_IS_LAST or Field 15: TOOL_IS_LAST_ALT
		if tcMsg.Has(FieldLegacyToolIsLast) {
			isLast = tcMsg.Get(FieldLegacyToolIsLast)[0].Varint != 0
		} else if tcMsg.Has(FieldLegacyToolIsLastAlt) {
			isLast = tcMsg.Get(FieldLegacyToolIsLastAlt)[0].Varint != 0
		}

		// Field 27: TOOL_MCP_PARAMS
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

		// Field 10: TOOL_RAW_ARGS
		if rawArgs == "" && tcMsg.Has(FieldLegacyToolRawArgs) {
			rawArgs = string(tcMsg.Get(FieldLegacyToolRawArgs)[0].Value)
		}

		if id != "" && name != "" {
			if rawArgs == "" {
				rawArgs = "{}"
			}
			res.ToolCall = &LegacyToolCall{
				ID:        id,
				Name:      name,
				Arguments: rawArgs,
				IsLast:    isLast,
			}
			return res
		}
	}

	// Field 2: StreamUnifiedChatResponse
	if fields.Has(FieldLegacyResponse) {
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
	}

	return res
}
