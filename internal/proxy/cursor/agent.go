package cursor

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// IsAgentCapableRequest checks if the request body messages contain only text content.
func IsAgentCapableRequest(body map[string]any) bool {
	rawMsgs, ok := body["messages"].([]any)
	if !ok || len(rawMsgs) == 0 {
		return false
	}
	for _, m := range rawMsgs {
		mMap, ok := m.(map[string]any)
		if !ok {
			return false
		}
		content := mMap["content"]
		if content == nil {
			continue
		}
		if _, isStr := content.(string); isStr {
			continue
		}
		if parts, isSlice := content.([]any); isSlice {
			for _, p := range parts {
				pMap, ok := p.(map[string]any)
				if !ok {
					if _, okStr := p.(string); okStr {
						continue
					}
					return false
				}
				pType, _ := pMap["type"].(string)
				if pType != "text" {
					return false
				}
			}
		} else {
			return false
		}
	}
	return true
}

func TextFromContent(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if parts, ok := content.([]any); ok {
		var sb strings.Builder
		for _, p := range parts {
			if pMap, ok := p.(map[string]any); ok {
				if pMap["type"] == "text" {
					if t, ok := pMap["text"].(string); ok {
						if sb.Len() > 0 {
							sb.WriteString("\n")
						}
						sb.WriteString(t)
					}
				}
			}
		}
		return sb.String()
	}
	return ""
}

func EncodeHistoryMessage(message map[string]any) []byte {
	role, _ := message["role"].(string)
	content := TextFromContent(message["content"])

	var extras []string
	if role == "assistant" {
		if tcs, ok := message["tool_calls"].([]any); ok {
			for _, tc := range tcs {
				if tcMap, ok := tc.(map[string]any); ok {
					id, _ := tcMap["id"].(string)
					fnName := "tool"
					args := "{}"
					if fn, ok := tcMap["function"].(map[string]any); ok {
						if n, ok := fn["name"].(string); ok {
							fnName = n
						}
						if a, ok := fn["arguments"].(string); ok {
							args = a
						}
					}
					extras = append(extras, fmt.Sprintf("[tool_call id=%s name=%s args=%s]", id, fnName, args))
				}
			}
		}
	}
	if role == "tool" {
		id, _ := message["tool_call_id"].(string)
		extras = append(extras, fmt.Sprintf("[tool_result id=%s]", id))
	}

	textBody := content
	if len(extras) > 0 {
		if textBody != "" {
			textBody += "\n" + strings.Join(extras, "\n")
		} else {
			textBody = strings.Join(extras, "\n")
		}
	}
	if textBody == "" {
		return nil
	}

	// ConversationHistoryMessage.user / .assistant -> repeated content -> text
	text := EncodeField(1, WireBytes, textBody)
	if role == "assistant" {
		return EncodeField(2, WireBytes, EncodeField(1, WireBytes, EncodeField(1, WireBytes, text)))
	}
	return EncodeField(1, WireBytes, EncodeField(1, WireBytes, EncodeField(1, WireBytes, text)))
}

// BuildAgentRunFrame builds the initial Connect-RPC AgentRunRequest frame.
func BuildAgentRunFrame(messages []any, model string, tools []any) []byte {
	var systemParts []string
	var chatMessages []map[string]any

	for _, m := range messages {
		mMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role, _ := mMap["role"].(string)
		if role == "system" {
			text := TextFromContent(mMap["content"])
			if text != "" {
				systemParts = append(systemParts, text)
			}
		} else {
			chatMessages = append(chatMessages, mMap)
		}
	}

	system := strings.Join(systemParts, "\n\n")
	currentIndex := -1
	for i := len(chatMessages) - 1; i >= 0; i-- {
		if r, _ := chatMessages[i]["role"].(string); r == "user" {
			currentIndex = i
			break
		}
	}

	var current map[string]any
	var historyMessages []map[string]any
	if currentIndex >= 0 {
		current = chatMessages[currentIndex]
		historyMessages = chatMessages[:currentIndex]
	} else if len(chatMessages) > 0 {
		current = chatMessages[len(chatMessages)-1]
		historyMessages = chatMessages[:len(chatMessages)-1]
	}

	var history [][]byte
	for _, h := range historyMessages {
		if enc := EncodeHistoryMessage(h); len(enc) > 0 {
			history = append(history, enc)
		}
	}

	rawUser := ""
	if current != nil {
		rawUser = TextFromContent(current["content"])
	}
	if rawUser == "" {
		rawUser = "Continue."
	}

	userText := rawUser
	if system != "" {
		userText = fmt.Sprintf("%s\n\n%s", system, rawUser)
	}

	userMessage := ConcatBuffers(
		EncodeField(1, WireBytes, userText),
		EncodeField(2, WireBytes, uuid.New().String()),
		EncodeField(3, WireBytes, []byte{}),
		EncodeField(4, WireVarint, 1),
	)

	var conversationHistory []byte
	if len(history) > 0 {
		var histEntries [][]byte
		for _, h := range history {
			histEntries = append(histEntries, EncodeField(1, WireBytes, h))
		}
		conversationHistory = ConcatBuffers(histEntries...)
	}

	var userActionParts [][]byte
	userActionParts = append(userActionParts, EncodeField(1, WireBytes, userMessage))
	if len(conversationHistory) > 0 {
		userActionParts = append(userActionParts, EncodeField(7, WireBytes, conversationHistory))
	}
	userAction := ConcatBuffers(userActionParts...)
	conversationAction := EncodeField(1, WireBytes, userAction)

	requestedModel := ConcatBuffers(
		EncodeField(1, WireBytes, model),
		EncodeField(7, WireVarint, 1),
	)

	modelDetails := ConcatBuffers(
		EncodeField(1, WireBytes, model),
		EncodeField(3, WireBytes, model),
		EncodeField(4, WireBytes, model),
	)

	mcpTools := EncodeMcpTools(tools)

	var runParts [][]byte
	runParts = append(runParts, EncodeField(1, WireBytes, []byte{}))
	runParts = append(runParts, EncodeField(2, WireBytes, conversationAction))
	runParts = append(runParts, EncodeField(3, WireBytes, modelDetails))
	if len(mcpTools) > 0 {
		runParts = append(runParts, EncodeField(4, WireBytes, mcpTools))
	}
	runParts = append(runParts, EncodeField(9, WireBytes, requestedModel))

	runRequest := ConcatBuffers(runParts...)
	agentClientMessage := EncodeField(1, WireBytes, runRequest)
	return WrapConnectRPCFrame(agentClientMessage)
}
