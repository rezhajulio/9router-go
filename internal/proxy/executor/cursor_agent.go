package executor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"9router/proxy/internal/proxy"

	cursorpkg "9router/proxy/internal/proxy/cursor"
)

// executeCursorAgent runs a turn against agent.api5.cursor.sh (AgentService),
// which is HTTP/2-only and speaks Connect-RPC bidi streaming.
func executeCursorAgent(w http.ResponseWriter, req *Request, bodyMap map[string]any, accessToken, machineID string, ghostMode bool) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	headers := cursorpkg.BuildCursorHeaders(accessToken, machineID, ghostMode)
	url := cursorAgentEndpoint + cursorAgentRunPath

	var tools []any
	if t, ok := bodyMap["tools"].([]any); ok {
		tools = t
	}

	model := req.ModelName
	if model == "" {
		if m, ok := bodyMap["model"].(string); ok {
			model = m
		}
	}
	model = stripCursorModelPrefix(model)

	rawMsgs, _ := bodyMap["messages"].([]any)
	runFrame := cursorpkg.BuildAgentRunFrame(rawMsgs, model, tools)

	session, err := openAgentHttp2Stream(ctx, url, headers, runFrame, req.Client)
	if err != nil {
		return err
	}
	defer session.Close()

	// Cancel the upstream session as soon as the client goes away. The stream
	// itself stays open on req.Ctx: an agent turn has no overall deadline.
	stop := context.AfterFunc(ctx, session.Close)
	defer stop()

	if session.resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(session.resp.Body, 1*1024*1024))
		message := fmt.Sprintf("Cursor AgentService %d: %s", session.resp.StatusCode,
			strings.ReplaceAll(string(errBody), `"`, `'`))
		// Report the upstream failure instead of writing a body: fallback needs
		// the status to refresh tokens and rotate combo accounts, and usage
		// logging needs to see the attempt fail.
		return &proxy.UpstreamError{
			StatusCode: session.resp.StatusCode,
			Body:       cursorErrorBody(message, "api_error"),
		}
	}

	isStream := req.IsStream
	if s, ok := bodyMap["stream"].(bool); ok {
		isStream = s
	}

	composerModel := cursorpkg.IsComposerModel(model)
	responseID := fmt.Sprintf("chatcmpl-msg_%d", time.Now().UnixMilli())
	created := time.Now().Unix()

	w = newCursorRecorderWriter(w, req)

	clientTools := cursorpkg.FilterClientToolsByChoice(cursorpkg.ParseClientTools(tools), bodyMap["tool_choice"])
	if isStream {
		return streamCursorAgent(w, req, session, model, composerModel, responseID, created, clientTools)
	}
	return respondCursorAgent(w, session, model, composerModel, responseID, created, req, clientTools)
}

// cursorClientPingInterval bounds how long the client-facing SSE stream may go
// without bytes on the wire. Upstream AgentService turns can legitimately stay
// quiet for minutes (long model latency, server-side tool work); without
// client-visible traffic harnesses with a stream watchdog (Amp reports "stream
// stalled after 120000 ms") kill a healthy turn. A var so tests can shrink it.
var cursorClientPingInterval = 20 * time.Second
