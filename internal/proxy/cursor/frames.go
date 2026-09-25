package cursor

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
)

// ConnectRPC compression flags
const (
	CompressFlagNone        = 0x00
	CompressFlagGzip        = 0x01
	CompressFlagTrailer     = 0x02
	CompressFlagGzipTrailer = 0x03
)

// DecompressPayload decompresses payload based on Connect-RPC compression flags.
func DecompressPayload(payload []byte, flags byte) ([]byte, error) {
	if len(payload) > 10 && payload[0] == '{' && payload[1] == '"' {
		if strings.HasPrefix(string(payload), "{\"error\"") {
			return payload, nil
		}
	}

	if flags == CompressFlagGzip || flags == CompressFlagTrailer || flags == CompressFlagGzipTrailer {
		// 1. Try gzip decompression
		gz, err := gzip.NewReader(bytes.NewReader(payload))
		if err == nil {
			decompressed, readErr := io.ReadAll(gz)
			_ = gz.Close()
			if readErr == nil {
				return decompressed, nil
			}
		}

		// 2. Try standard zlib decompression (RFC 1950 header)
		zr, err := zlib.NewReader(bytes.NewReader(payload))
		if err == nil {
			decompressed, readErr := io.ReadAll(zr)
			_ = zr.Close()
			if readErr == nil {
				return decompressed, nil
			}
		}

		// 3. Fall back to raw deflate (RFC 1951 without wrapper headers)
		fl := flate.NewReader(bytes.NewReader(payload))
		decompressed, err := io.ReadAll(fl)
		_ = fl.Close()
		if err == nil {
			return decompressed, nil
		}

		return payload, nil
	}

	return payload, nil
}

// DecodeAgentFrames reads Connect-RPC frames from buffer and calls onFrame for each payload.
// Returns remaining unconsumed bytes.
func DecodeAgentFrames(buffer []byte, onFrame func(payload []byte)) []byte {
	pending := buffer
	for len(pending) >= 5 {
		flags := pending[0]
		length := binary.BigEndian.Uint32(pending[1:5])
		frameLen := 5 + int(length)
		if len(pending) < frameLen {
			break
		}

		payload := pending[5:frameLen]
		pending = pending[frameLen:]

		decompressed, err := DecompressPayload(payload, flags)
		if err == nil && (flags&CompressFlagTrailer) == 0 {
			onFrame(decompressed)
		}
	}
	return pending
}

// WrapExecClientMessage wraps an ExecClientMessage response into Connect-RPC frame.
func WrapExecClientMessage(execMsgID uint64, execID string, resultField int, resultPayload []byte) []byte {
	var parts [][]byte
	if execMsgID > 0 {
		parts = append(parts, EncodeField(1, WireVarint, execMsgID))
	}
	parts = append(parts, EncodeField(15, WireBytes, execID))
	parts = append(parts, EncodeField(resultField, WireBytes, resultPayload))

	execClientMsg := ConcatBuffers(parts...)
	agentClientMsg := EncodeField(2, WireBytes, execClientMsg)
	return WrapConnectRPCFrame(agentClientMsg)
}

// CreateRequestContextResponse acknowledges a request_context query from Cursor AgentService.
func CreateRequestContextResponse(execRequest DecodedMessage) []byte {
	var id uint64
	if execRequest.Has(1) {
		id = execRequest.Get(1)[0].Varint
	}
	execID := ""
	if execRequest.Has(15) {
		execID = string(execRequest.Get(15)[0].Value)
	}

	requestContextSuccess := EncodeField(1, WireBytes, []byte{})
	requestContextResult := EncodeField(1, WireBytes, requestContextSuccess)
	return WrapExecClientMessage(id, execID, 10, requestContextResult)
}

// RejectExecRequest rejects unknown IDE builtins so the agent continues with text or MCP tools.
var execResultFields = map[int]int{
	2: 2, 3: 3, 4: 4, 5: 5, 7: 7, 8: 8, 9: 9, 16: 16, 20: 20, 23: 23,
}

func RejectExecRequest(execRequest DecodedMessage) []byte {
	var id uint64
	if execRequest.Has(1) {
		id = execRequest.Get(1)[0].Varint
	}
	execID := ""
	if execRequest.Has(15) {
		execID = string(execRequest.Get(15)[0].Value)
	}

	variant := 0
	for _, k := range execRequest.Keys() {
		if k != 1 && k != 15 {
			variant = k
			break
		}
	}

	resultField, ok := execResultFields[variant]
	if !ok {
		return nil
	}

	if variant == 9 {
		// Diagnostics has no rejected variant — empty success unblocks the stream
		return WrapExecClientMessage(id, execID, 9, []byte{})
	}

	rejected := EncodeField(2, WireBytes, EncodeField(2, WireBytes, "Tool not available in this environment. Use the MCP tools provided instead."))
	return WrapExecClientMessage(id, execID, resultField, rejected)
}

// EncodeKvClientMessage responds to KvServerMessage blob requests.
func EncodeKvClientMessage(kvID uint64, resultField int, resultPayload []byte, metadata []byte) []byte {
	var parts [][]byte
	if kvID > 0 {
		parts = append(parts, EncodeField(1, WireVarint, kvID))
	}
	parts = append(parts, EncodeField(resultField, WireBytes, resultPayload))
	if len(metadata) > 0 {
		parts = append(parts, EncodeField(4, WireBytes, metadata))
	}

	kvClientMsg := ConcatBuffers(parts...)
	agentClientMsg := EncodeField(3, WireBytes, kvClientMsg)
	return WrapConnectRPCFrame(agentClientMsg)
}

// IsComposerModel returns true if model is a Composer variant.
func IsComposerModel(model string) bool {
	parts := strings.Split(model, "/")
	modelID := strings.ToLower(parts[len(parts)-1])
	return modelID == "composer" || strings.HasPrefix(modelID, "composer-")
}

// VisibleComposerContentFromThinking extracts visible response after </think>.
func VisibleComposerContentFromThinking(thinking string) string {
	if thinking == "" {
		return ""
	}
	endTag := "</think>"
	idx := strings.LastIndex(thinking, endTag)
	if idx < 0 {
		return ""
	}
	return strings.TrimLeft(thinking[idx+len(endTag):], " \t\r\n")
}
