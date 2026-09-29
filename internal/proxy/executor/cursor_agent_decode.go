package executor

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// agentEventKind classifies one decoded AgentServerMessage event.
type agentEventKind int

const (
	agentEventText agentEventKind = iota
	agentEventThinking
	agentEventToolCall
	agentEventTurnEnded
)

// agentToolCall is a tool call the client has to execute.
type agentToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// agentEvent is what one upstream payload means for the client.
type agentEvent struct {
	Kind     agentEventKind
	Text     string
	ToolCall agentToolCall
}

// terminal reports whether processing of the payload must stop here. A tool call
// and a turn end both end the relay, so a tool call from the same frame can
// never be written after the terminal chunk.
func (k agentEventKind) terminal() bool {
	return k == agentEventToolCall || k == agentEventTurnEnded
}

func lastIsTerminal(events []agentEvent) bool {
	return len(events) > 0 && events[len(events)-1].Kind.terminal()
}

// decodeAgentPayload converts one AgentServerMessage payload into the events the
// client should see, answering the upstream control frames (kv blob get/set,
// request_context, exec rejections) as a side effect.
//
// The streaming and the non-streaming relay both go through this, so the frame
// dispatch cannot drift between them. The order matches the wire: interaction
// deltas first, then the kv exchange, then the exec request; once a terminal
// event is produced the rest of the payload is skipped exactly as the previous
// per-path callbacks did.
func decodeAgentPayload(session *agentSession, clientTools []cursorpkg.ClientTool, payload []byte) []agentEvent {
	serverMsg := cursorpkg.DecodeMessage(payload)

	var events []agentEvent
	if serverMsg.Has(cursorpkg.FieldInteractionUpdate) {
		update := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldInteractionUpdate)[0].Value)
		if update.Has(cursorpkg.FieldTextDelta) {
			sub := cursorpkg.DecodeMessage(update.Get(cursorpkg.FieldTextDelta)[0].Value)
			if sub.Has(1) {
				if delta := string(sub.Get(1)[0].Value); delta != "" {
					events = append(events, agentEvent{Kind: agentEventText, Text: delta})
				}
			}
		}
		if update.Has(cursorpkg.FieldThinkingDelta) {
			sub := cursorpkg.DecodeMessage(update.Get(cursorpkg.FieldThinkingDelta)[0].Value)
			if sub.Has(1) {
				if delta := string(sub.Get(1)[0].Value); delta != "" {
					events = append(events, agentEvent{Kind: agentEventThinking, Text: delta})
				}
			}
		}
		if update.Has(cursorpkg.FieldTurnEnded) {
			events = append(events, agentEvent{Kind: agentEventTurnEnded})
		}
		if lastIsTerminal(events) {
			return events
		}
	}

	// kv_server_message (field 4): blob get/set. Ack so the stream proceeds.
	if serverMsg.Has(cursorpkg.FieldKvServerMessage) {
		kv := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldKvServerMessage)[0].Value)
		var kvID uint64
		if kv.Has(1) {
			kvID = kv.Get(1)[0].Varint
		}
		var meta []byte
		if kv.Has(4) {
			meta = kv.Get(4)[0].Value
		}
		if kv.Has(2) {
			_ = session.Write(cursorpkg.EncodeKvClientMessage(kvID, 2, cursorpkg.EncodeField(1, cursorpkg.WireBytes, []byte{}), meta))
		} else if kv.Has(3) {
			_ = session.Write(cursorpkg.EncodeKvClientMessage(kvID, 3, []byte{}, meta))
		}
	}

	// exec_request (field 2)
	if serverMsg.Has(cursorpkg.FieldExecRequest) {
		execReq := cursorpkg.DecodeMessage(serverMsg.Get(cursorpkg.FieldExecRequest)[0].Value)
		if call, ok := handleExecRequest(session, clientTools, execReq); ok {
			events = append(events, agentEvent{Kind: agentEventToolCall, ToolCall: call})
		}
	}

	return events
}

// handleExecRequest answers one exec request. It returns the tool call the
// client has to execute, and ok=false for every request that only needed an
// upstream answer (request_context ack, kv-style controls, a declined tool).
func handleExecRequest(session *agentSession, clientTools []cursorpkg.ClientTool, execReq cursorpkg.DecodedMessage) (agentToolCall, bool) {
	switch {
	case execReq.Has(10):
		_ = session.Write(cursorpkg.CreateRequestContextResponse(execReq))
		return agentToolCall{}, false

	case execReq.Has(11):
		mcp := cursorpkg.DecodeMcpArgs(execReq.Get(11)[0].Value)
		name := mcp.ToolName
		if name == "" {
			name = mcp.Name
		}
		if name == "" || mcp.Truncated {
			// No name, or arguments that did not decode cleanly: nothing safe can
			// be handed to the client. Decline the tool so the server keeps the
			// turn alive instead of the stream failing, or a client tool acting
			// on partial input.
			writeExecControlFrames(session, execReq, "Cursor AgentService requested an MCP tool without a name", "exec_variant_unsupported")
			return agentToolCall{}, false
		}
		id := mcp.ToolCallID
		if id == "" {
			id = fmt.Sprintf("call_%s", uuid.New().String())
		}
		return agentToolCall{ID: id, Name: name, Args: mcp.Args}, true

	default:
		// OmniRoute parity: a native IDE builtin (shell/read) the model asked for
		// is bridged onto a schema-compatible client tool when one exists,
		// instead of only being declined. The typed rejection still goes
		// upstream; the bridged call is emitted as a normal OpenAI tool_call so
		// the harness can execute it and return the result on the next turn.
		if bridge := cursorpkg.BridgeBuiltinTool(cursorpkg.DecodeBuiltinEvent(execReq), clientTools); bridge != nil {
			rejectExecRequest(session, execReq)
			return agentToolCall{
				ID:   fmt.Sprintf("call_%s", uuid.New().String()),
				Name: bridge.ToolName,
				Args: bridge.Arguments,
			}, true
		}
		// A Cursor IDE builtin this proxy cannot execute. A typed rejection
		// already tells the model that; a variant without one gets a throw plus
		// the matching stream close, which keeps the turn going (this is how omp
		// answers variants it has no handler for) whereas ending the stream
		// failed the whole request.
		rejectExecRequest(session, execReq)
		return agentToolCall{}, false
	}
}

// rejectExecRequest answers an exec request the proxy will not execute: the
// typed rejection when the variant has one, otherwise a throw plus stream close.
func rejectExecRequest(session *agentSession, execReq cursorpkg.DecodedMessage) {
	if rej := cursorpkg.RejectExecRequest(execReq); rej != nil {
		_ = session.Write(rej)
		return
	}
	writeExecControlFrames(session, execReq,
		fmt.Sprintf("Cursor AgentService requested IDE tool variant %d, which this proxy cannot execute", cursorpkg.ExecRequestVariant(execReq)),
		"exec_variant_unsupported")
}

// agentToolCallJSON renders a tool call's arguments the way the OpenAI wire
// format carries them: an empty object, never a JSON null, even when the
// upstream sent no arguments at all.
func agentToolCallJSON(call agentToolCall) string {
	if len(call.Args) == 0 {
		return "{}"
	}
	b, err := json.Marshal(call.Args)
	if err != nil {
		return "{}"
	}
	return string(b)
}
