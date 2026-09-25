package cursor

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
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

// maxDecompressedFrameLength caps decompressed output: the compressed frame is
// already size-checked, but deflate ratios are unbounded, so a small hostile
// payload could otherwise expand until the proxy runs out of memory.
const maxDecompressedFrameLength = 64 << 20

// readBoundedFrame reads a decompressed frame up to maxDecompressedFrameLength.
// Exceeding the cap is an error rather than a silent truncation.
func readBoundedFrame(r io.Reader) ([]byte, error) {
	out, err := io.ReadAll(io.LimitReader(r, maxDecompressedFrameLength+1))
	if err != nil {
		return nil, err
	}
	if len(out) > maxDecompressedFrameLength {
		return nil, fmt.Errorf("decompressed frame exceeds %d bytes", maxDecompressedFrameLength)
	}
	return out, nil
}

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
			decompressed, readErr := readBoundedFrame(gz)
			_ = gz.Close()
			if readErr == nil {
				return decompressed, nil
			}
		}

		// 2. Try standard zlib decompression (RFC 1950 header)
		zr, err := zlib.NewReader(bytes.NewReader(payload))
		if err == nil {
			decompressed, readErr := readBoundedFrame(zr)
			_ = zr.Close()
			if readErr == nil {
				return decompressed, nil
			}
		}

		// 3. Fall back to raw deflate (RFC 1951 without wrapper headers)
		fl := flate.NewReader(bytes.NewReader(payload))
		decompressed, err := readBoundedFrame(fl)
		_ = fl.Close()
		if err == nil {
			return decompressed, nil
		}

		return payload, nil
	}

	return payload, nil
}

// maxAgentFrameLength caps a single Connect-RPC frame. Real Cursor frames are a
// few KiB; the cap only exists so a corrupt or hostile frame header cannot keep
// DecodeAgentFrames buffering for a frame that will never complete.
const maxAgentFrameLength = 8 << 20

// DecodeAgentFrames reads Connect-RPC frames from buffer and calls onFrame for
// each payload. It returns the remaining unconsumed bytes, and ok=false when a
// frame declared a length above maxAgentFrameLength: the stream cannot be
// resynchronised past that frame, so callers must stop reading.
func DecodeAgentFrames(buffer []byte, onFrame func(payload []byte)) (pending []byte, ok bool) {
	pending = buffer
	for len(pending) >= 5 {
		flags := pending[0]
		length := binary.BigEndian.Uint32(pending[1:5])
		if length > maxAgentFrameLength {
			return pending, false
		}
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
	return pending, true
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

// RejectExecRequest rejects IDE builtins this proxy does not execute so the
// agent continues with text or MCP tools. The result payload carries the same
// field number as the requested variant. Returns nil when the variant has no
// known rejected shape, in which case callers answer with ExecClientControlFrames.
func RejectExecRequest(execRequest DecodedMessage) []byte {
	id := execRequestID(execRequest)
	execID := ""
	if execRequest.Has(15) {
		execID = string(execRequest.Get(15)[0].Value)
	}

	variant := ExecRequestVariant(execRequest)
	if variant == 0 || !execVariantFields[variant] {
		return nil
	}
	resultField := variant

	if variant == 9 {
		// Diagnostics has no rejected variant — empty success unblocks the stream
		return WrapExecClientMessage(id, execID, 9, []byte{})
	}

	rejected := EncodeField(2, WireBytes, EncodeField(2, WireBytes, "Tool not available in this environment. Use the MCP tools provided instead."))
	return WrapExecClientMessage(id, execID, resultField, rejected)
}

// execVariantFields are the ExecServerMessage oneof field numbers whose result
// message accepts the generic rejected payload built by RejectExecRequest (a
// refusal string under field 2, or the empty success for 9).
//
// Every other variant — including 3/4/7/8/16/20 (Write/Delete/Read/Ls/
// BackgroundShellSpawn/Fetch), whose result messages put the refusal under 3 or
// 6 or expect a typed message, and the newer CLI variants (27-31, 37-38, 40-55)
// — is answered with ExecClientControlFrames instead. Writing the generic blob
// there would decode as the variant's *error* or *success* field and hand the
// model a corrupted tool result, so a throw is both safer and what omp does for
// variants it has no typed handler for.
var execVariantFields = map[int]bool{
	2: true, 5: true, 9: true, 23: true, 36: true,
}

// ExecClientControlFrames builds the two client frames that decline to execute
// an IDE builtin: a throw naming the variant, then a stream close.
//
// AgentClientMessage.execClientControlMessage is field 5; within it throw = 2
// (id = 1, error = 2, errorCode = 4) and streamClose = 1 (id = 1), matching
// agent.v1.ExecClientControlMessage. The server reads the throw as "the client
// could not run this tool" and keeps the turn going, so the model can fall back
// to MCP tools or a text answer instead of the turn dying with an error.
func ExecClientControlFrames(execRequest DecodedMessage, errMessage, errCode string) [][]byte {
	id := execRequestID(execRequest)

	throw := ConcatBuffers(
		EncodeField(1, WireVarint, uint64(id)),
		EncodeField(2, WireBytes, []byte(errMessage)),
		EncodeField(4, WireBytes, []byte(errCode)),
	)
	control := EncodeField(2, WireBytes, throw)

	streamClose := EncodeField(1, WireBytes, EncodeField(1, WireVarint, uint64(id)))

	return [][]byte{
		WrapConnectRPCFrame(EncodeField(5, WireBytes, control)),
		WrapConnectRPCFrame(EncodeField(5, WireBytes, streamClose)),
	}
}

// execRequestID reads ExecServerMessage.id (field 1).
func execRequestID(execRequest DecodedMessage) uint64 {
	if execRequest.Has(1) {
		return execRequest.Get(1)[0].Varint
	}
	return 0
}

// ExecRequestVariant returns the oneof field number identifying the requested
// IDE tool. Field 1 is the message id, 15 is exec_id, and 19/55 carry
// trace/context metadata (RequestTracingData / flags) that arrive alongside the
// variant, so none of them identify the tool. Every ExecServerMessage variant is
// a message, so a length-delimited field is required; anything else (a malformed
// or hostile frame) yields 0 and lands in the throw path.
func ExecRequestVariant(execRequest DecodedMessage) int {
	for _, k := range execRequest.Keys() {
		if k == 1 || k == 15 || k == 19 || k == 55 {
			continue
		}
		if execRequest.Get(k)[0].WireType != WireBytes {
			continue
		}
		return k
	}
	return 0
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
