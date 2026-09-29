package cursor

import (
	"fmt"
	"runtime"
	"strings"
	"time"
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
	FieldLegacyToolCall        = 1
	FieldLegacyResponse        = 2
	FieldLegacyToolID          = 3
	FieldLegacyToolName        = 9
	FieldLegacyToolRawArgs     = 10
	FieldLegacyToolIsLast      = 11
	FieldLegacyToolIsLastAlt   = 15
	FieldLegacyToolMCPParams   = 27
	FieldLegacyMCPToolsList    = 1
	FieldLegacyMCPNestedName   = 1
	FieldLegacyMCPNestedParams = 3
	FieldLegacyResponseText    = 1
	FieldLegacyThinking        = 25
	FieldLegacyThinkingText    = 1

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

	// McpParams nested (name / raw args / server)
	FieldMCPParamsName    = 1
	FieldMCPParamsRawArgs = 2
	FieldMCPParamsServer  = 3

	// MCP tool declaration, matching upstream encodeMcpTool's field order.
	FieldMcpDeclName   = 1
	FieldMcpDeclDesc   = 2
	FieldMcpDeclSchema = 3
	FieldMcpDeclServer = 4
)

// SystemInstructionPrefix marks a system message that travelled as a user
// bubble, matching upstream's openai-to-cursor convertMessages.
const SystemInstructionPrefix = "[System Instructions]\n"

// LegacyToolCall holds tool call extracted from ChatService response.
type LegacyToolCall struct {
	ID        string
	Name      string
	Arguments string
	IsLast    bool
}

// LegacyResponseFrame holds parsed content from ChatService response frame.
type LegacyResponseFrame struct {
	Text      string
	Thinking  string
	ToolCall  *LegacyToolCall
	Error     string
	ErrorCode string
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
		EncodeField(FieldMCPParamsName, WireBytes, toolName),
		EncodeField(FieldMCPParamsRawArgs, WireBytes, rawArgs),
		EncodeField(FieldMCPParamsServer, WireBytes, serverName),
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
	switch plat {
	case "windows":
		plat = "win32"
	case "darwin", "linux":
		// already matches Node process.platform
	default:
		plat = "linux"
	}
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "ia32"
	}
	return EncodeField(FieldLegacyMetadata, WireBytes, ConcatBuffers(
		EncodeField(1, WireBytes, plat),
		EncodeField(2, WireBytes, arch),
		EncodeField(3, WireBytes, "v20.0.0"),
		EncodeField(4, WireBytes, "/"),
		EncodeField(5, WireBytes, time.Now().UTC().Format("2006-01-02T15:04:05.000Z")),
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
