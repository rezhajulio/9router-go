package executor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// streamCursorAgent relays an AgentService turn as OpenAI SSE chunks.
func streamCursorAgent(w http.ResponseWriter, req *Request, session *agentSession, model string, composerModel bool, responseID string, created int64, clientTools []cursorpkg.ClientTool) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	var pending []byte
	var thinkingAcc strings.Builder
	emittedVisible := 0
	finished := false
	emittedText := false
	emittedReasoning := false
	streamErr := ""
	toolIndex := 0
	completionChars := 0
	// Published on every exit path, error frames included.
	defer func() { recordCursorUsage(req, completionChars) }()

	// Reads run on a separate goroutine so the ticker below can emit SSE
	// comment keepalives while upstream is quiet; blocking on ReadChunk
	// directly would leave the client stream silent until the turn ends.
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	type cursorReadResult struct {
		chunk []byte
		err   error
	}
	doneCh := make(chan struct{})
	defer close(doneCh)
	reads := make(chan cursorReadResult, 1)
	go func() {
		defer close(reads)
		for {
			chunk, readErr := session.ReadChunk()
			// ReadChunk reuses its buffer across calls, so the bytes must be
			// copied before they cross to the relay loop.
			var cp []byte
			if len(chunk) > 0 {
				cp = append([]byte(nil), chunk...)
			}
			select {
			case reads <- cursorReadResult{chunk: cp, err: readErr}:
			case <-ctx.Done():
				return
			case <-doneCh:
				return
			}
			if readErr != nil {
				return
			}
		}
	}()
	ping := time.NewTicker(cursorClientPingInterval)
	defer ping.Stop()
	lastActivity := time.Now()
	// Keepalives bypass the recorder wrapper: comment bytes must not pollute
	// the mirrored response buffer or trip the TTFT measurement.
	pingSent := false
	pingWriter := w
	if rw, ok := w.(*cursorRecorderWriter); ok {
		pingWriter = rw.ResponseWriter
	}

relay:
	for !finished && streamErr == "" {
		select {
		case <-ctx.Done():
			break relay
		case <-ping.C:
			if time.Since(lastActivity) >= cursorClientPingInterval {
				_, _ = fmt.Fprint(pingWriter, ": ping\n\n")
				if flusher != nil {
					flusher.Flush()
				}
				pingSent = true
			}
		case res, alive := <-reads:
			if !alive {
				break relay
			}
			chunk, readErr := res.chunk, res.err
			if len(chunk) > 0 {
				lastActivity = time.Now()
				pending = append(pending, chunk...)
				var ok bool
				pending, ok = cursorpkg.DecodeAgentFrames(pending, func(payload []byte) {
					if finished {
						return
					}
					for _, ev := range decodeAgentPayload(session, clientTools, payload) {
						switch ev.Kind {
						case agentEventText:
							emittedText = true
							completionChars += len(ev.Text)
							writeSSEChunk(w, flusher, responseID, created, model, ev.Text, nil, "")

						case agentEventThinking:
							if composerModel {
								thinkingAcc.WriteString(ev.Text)
								vis := cursorpkg.VisibleComposerContentFromThinking(thinkingAcc.String())
								if len(vis) > emittedVisible {
									delta := vis[emittedVisible:]
									emittedVisible = len(vis)
									emittedText = true
									completionChars += len(delta)
									writeSSEChunk(w, flusher, responseID, created, model, delta, nil, "")
								}
							} else {
								// Non-composer models stream thinking as reasoning_content.
								emittedReasoning = true
								completionChars += len(ev.Text)
								writeSSEReasoningChunk(w, flusher, responseID, created, model, ev.Text)
							}

						case agentEventToolCall:
							tcID := ev.ToolCall.ID
							if tcID == "" {
								tcID = fmt.Sprintf("call_%s", uuid.New().String())
							}
							args := agentToolCallJSON(ev.ToolCall)
							completionChars += len(args)
							writeSSEToolCall(w, flusher, responseID, created, model, tcID, ev.ToolCall.Name, args, toolIndex)
							toolIndex++
							finished = true
							writeAgentTerminal(w, flusher, req, responseID, created, model, toolIndex, completionChars)
							return

						case agentEventTurnEnded:
							if !emittedText && thinkingAcc.Len() > 0 && composerModel {
								fb := cursorpkg.VisibleComposerContentFromThinking(thinkingAcc.String())
								if fb != "" {
									completionChars += len(fb)
									writeSSEChunk(w, flusher, responseID, created, model, fb, nil, "")
								}
							}
							finished = true
							// The turn is over and its terminal chunk is out: nothing
							// else in this payload may be handled, or a tool call from
							// the same frame would be written after [DONE].
							writeAgentTerminal(w, flusher, req, responseID, created, model, toolIndex, completionChars)
							return
						}
					}
				})
				if !ok {
					streamErr = "Cursor AgentService frame exceeded the accepted size"
				}
			}
			if readErr != nil {
				break relay
			}
		}
	}

	// finished means the turn_ended terminal (finish chunk + [DONE]) is already on
	// the wire, so nothing else may be written for this request — not an error
	// frame, and above all not a second [DONE].
	if finished {
		return nil
	}

	if ctx.Err() != nil {
		if pingSent || emittedText || toolIndex > 0 {
			return &errCursorCommitted{err: ctx.Err()}
		}
		return ctx.Err()
	}

	if pingSent {
		msg := streamErr
		if msg == "" {
			msg = "Cursor AgentService stream closed before the turn ended"
		}
		writeSSEError(w, flusher, msg)
		return &errCursorCommitted{err: errors.New(msg)}
	}
	if streamErr != "" && !emittedText && !emittedReasoning && thinkingAcc.Len() == 0 && toolIndex == 0 {
		// Nothing was written yet, so this can still be reported as a failed
		// attempt instead of a truncated SSE stream the caller would log as a
		// success. The only streaming failure left here is a framing one (an
		// oversized frame), hence 502.
		return &proxy.UpstreamError{StatusCode: http.StatusBadGateway, Body: cursorErrorBody(streamErr, "api_error")}
	}
	if streamErr != "" {
		writeSSEError(w, flusher, streamErr)
		return nil
	}

	if !emittedText && !emittedReasoning && thinkingAcc.Len() == 0 && toolIndex == 0 {
		// The connection ended before the turn did and produced nothing: reporting
		// a finish_reason stop here would present a dead stream as a complete empty
		// answer and log the attempt as a success.
		return &proxy.UpstreamError{
			StatusCode: http.StatusBadGateway,
			Body:       cursorErrorBody("Cursor AgentService stream closed before the turn ended", "api_error"),
		}
	}

	if !emittedText && thinkingAcc.Len() > 0 && composerModel {
		fb := cursorpkg.VisibleComposerContentFromThinking(thinkingAcc.String())
		if fb != "" {
			completionChars += len(fb)
			writeSSEChunk(w, flusher, responseID, created, model, fb, nil, "")
		}
	}
	writeAgentTerminal(w, flusher, req, responseID, created, model, toolIndex, completionChars)

	return nil
}

// writeAgentTerminal writes the single terminal chunk for an agent turn: the
// finish_reason chunk carrying the estimated usage, then [DONE]. Every exit path
// that ends a turn normally goes through here, so the stream can only ever
// terminate once.
func writeAgentTerminal(w http.ResponseWriter, flusher http.Flusher, req *Request, responseID string, created int64, model string, toolCalls, completionChars int) {
	promptTokens := len(req.Body) / 4
	compTokens := completionChars / 4
	usageMap := map[string]any{
		"prompt_tokens":     promptTokens,
		"completion_tokens": compTokens,
		"total_tokens":      promptTokens + compTokens,
	}
	writeSSEChunkWithUsage(w, flusher, responseID, created, model, "", nil, finishReasonFor(toolCalls), usageMap)
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// writeExecControlFrames declines an IDE builtin: a throw naming the variant plus
// the matching stream close, so the server resumes the turn instead of the
// client having to kill the stream.
func writeExecControlFrames(session *agentSession, execReq cursorpkg.DecodedMessage, message, code string) {
	for _, frame := range cursorpkg.ExecClientControlFrames(execReq, message, code) {
		_ = session.Write(frame)
	}
}
