package cursor

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
)

// Wire types
const (
	WireVarint  = 0
	WireFixed64 = 1
	WireBytes   = 2
	WireFixed32 = 5
)

// Agent / Protobuf field constants
const (
	FieldRunRequest          = 1
	FieldAction              = 2
	FieldModelDetails        = 3
	FieldMcpTools            = 4
	FieldRequestedModel      = 9
	FieldInteractionUpdate   = 1
	FieldExecRequest         = 2
	FieldKvServerMessage     = 4
	FieldTextDelta           = 1
	FieldThinkingDelta       = 4
	FieldTurnEnded           = 14
)

// Value types for google.protobuf.Value
const (
	PbValueNull   = 1
	PbValueNumber = 2
	PbValueString = 3
	PbValueBool   = 4
	PbValueStruct = 5
	PbValueList   = 6
)

// Field maps: tag = (fieldNum << 3) | wireType
func EncodeVarint(val uint64) []byte {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], val)
	return buf[:n]
}

func EncodeField(fieldNum int, wireType int, val any) []byte {
	tag := EncodeVarint(uint64((fieldNum << 3) | wireType))
	var data []byte

	switch wireType {
	case WireVarint:
		var v uint64
		switch n := val.(type) {
		case int:
			v = uint64(n)
		case int32:
			v = uint64(n)
		case int64:
			v = uint64(n)
		case uint:
			v = uint64(n)
		case uint32:
			v = uint64(n)
		case uint64:
			v = n
		case bool:
			if n {
				v = 1
			} else {
				v = 0
			}
		}
		data = EncodeVarint(v)

	case WireFixed64:
		var buf [8]byte
		switch n := val.(type) {
		case float64:
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(n))
		case uint64:
			binary.LittleEndian.PutUint64(buf[:], n)
		case int64:
			binary.LittleEndian.PutUint64(buf[:], uint64(n))
		}
		data = buf[:]

	case WireFixed32:
		var buf [4]byte
		switch n := val.(type) {
		case float32:
			binary.LittleEndian.PutUint32(buf[:], math.Float32bits(n))
		case uint32:
			binary.LittleEndian.PutUint32(buf[:], n)
		case int32:
			binary.LittleEndian.PutUint32(buf[:], uint32(n))
		}
		data = buf[:]

	case WireBytes:
		var b []byte
		switch s := val.(type) {
		case string:
			b = []byte(s)
		case []byte:
			b = s
		}
		length := EncodeVarint(uint64(len(b)))
		data = append(length, b...)
	}

	return append(tag, data...)
}

func ConcatBuffers(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	res := make([]byte, 0, total)
	for _, p := range parts {
		res = append(res, p...)
	}
	return res
}

// WrapConnectRPCFrame wraps a protobuf payload in a 5-byte Connect-RPC frame.
func WrapConnectRPCFrame(payload []byte, compress ...bool) []byte {
	useGzip := len(compress) > 0 && compress[0]
	finalPayload := payload
	flags := byte(0x00)

	if useGzip {
		var gzBuf bytes.Buffer
		w := gzip.NewWriter(&gzBuf)
		_, _ = w.Write(payload)
		_ = w.Close()
		finalPayload = gzBuf.Bytes()
		flags = 0x01
	}

	frame := make([]byte, 5+len(finalPayload))
	frame[0] = flags
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(finalPayload)))
	copy(frame[5:], finalPayload)
	return frame
}

// DecodedField represents a single protobuf field.
type DecodedField struct {
	WireType int
	Value    []byte
	Varint   uint64
}

// DecodedMessage maps field number to repeated values.
type DecodedMessage map[int][]DecodedField

func (m DecodedMessage) Has(fieldNum int) bool {
	return len(m[fieldNum]) > 0
}

func (m DecodedMessage) Get(fieldNum int) []DecodedField {
	return m[fieldNum]
}

func (m DecodedMessage) Keys() []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func DecodeVarint(buffer []byte, offset int) (uint64, int, error) {
	val, n := binary.Uvarint(buffer[offset:])
	if n <= 0 {
		return 0, offset, fmt.Errorf("invalid varint at offset %d", offset)
	}
	return val, offset + n, nil
}

func DecodeField(buffer []byte, offset int) (fieldNum int, wireType int, field DecodedField, newOffset int, err error) {
	if offset >= len(buffer) {
		return 0, 0, DecodedField{}, offset, io.EOF
	}

	tag, pos, err := DecodeVarint(buffer, offset)
	if err != nil {
		return 0, 0, DecodedField{}, offset, err
	}

	fieldNum = int(tag >> 3)
	wireType = int(tag & 0x07)

	switch wireType {
	case WireVarint:
		v, nextPos, err := DecodeVarint(buffer, pos)
		if err != nil {
			return 0, 0, DecodedField{}, offset, err
		}
		field = DecodedField{WireType: wireType, Varint: v}
		newOffset = nextPos

	case WireBytes:
		length, nextPos, err := DecodeVarint(buffer, pos)
		if err != nil {
			return 0, 0, DecodedField{}, offset, err
		}
		end := nextPos + int(length)
		if end > len(buffer) {
			return 0, 0, DecodedField{}, offset, fmt.Errorf("length delimiter exceeds buffer")
		}
		field = DecodedField{WireType: wireType, Value: buffer[nextPos:end]}
		newOffset = end

	case WireFixed64:
		if pos+8 > len(buffer) {
			return 0, 0, DecodedField{}, offset, fmt.Errorf("fixed64 exceeds buffer")
		}
		field = DecodedField{WireType: wireType, Value: buffer[pos : pos+8]}
		newOffset = pos + 8

	case WireFixed32:
		if pos+4 > len(buffer) {
			return 0, 0, DecodedField{}, offset, fmt.Errorf("fixed32 exceeds buffer")
		}
		field = DecodedField{WireType: wireType, Value: buffer[pos : pos+4]}
		newOffset = pos + 4

	default:
		return 0, 0, DecodedField{}, offset, fmt.Errorf("unsupported wire type: %d", wireType)
	}

	return fieldNum, wireType, field, newOffset, nil
}

func DecodeMessage(data []byte) DecodedMessage {
	fields := make(DecodedMessage)
	pos := 0

	for pos < len(data) {
		fNum, _, field, nextPos, err := DecodeField(data, pos)
		if err != nil {
			break
		}
		fields[fNum] = append(fields[fNum], field)
		pos = nextPos
	}

	return fields
}

// EncodeAgentValue encodes a Go value as google.protobuf.Value.
func EncodeAgentValue(value any) []byte {
	if value == nil {
		return EncodeField(PbValueNull, WireVarint, 0)
	}

	switch v := value.(type) {
	case bool:
		bVal := 0
		if v {
			bVal = 1
		}
		return EncodeField(PbValueBool, WireVarint, bVal)
	case int:
		return EncodeField(PbValueNumber, WireFixed64, float64(v))
	case int64:
		return EncodeField(PbValueNumber, WireFixed64, float64(v))
	case float64:
		return EncodeField(PbValueNumber, WireFixed64, v)
	case string:
		return EncodeField(PbValueString, WireBytes, v)
	case []any:
		var items [][]byte
		for _, item := range v {
			items = append(items, EncodeField(1, WireBytes, EncodeAgentValue(item)))
		}
		return EncodeField(PbValueList, WireBytes, ConcatBuffers(items...))
	case map[string]any:
		// Sort keys for deterministic protobuf bytes
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var entries [][]byte
		for _, k := range keys {
			val := v[k]
			entry := ConcatBuffers(
				EncodeField(1, WireBytes, k),
				EncodeField(2, WireBytes, EncodeAgentValue(val)),
			)
			entries = append(entries, EncodeField(1, WireBytes, entry))
		}
		return EncodeField(PbValueStruct, WireBytes, ConcatBuffers(entries...))
	default:
		return EncodeField(PbValueString, WireBytes, fmt.Sprintf("%v", v))
	}
}

// DecodeAgentValue decodes google.protobuf.Value bytes to Go any.
func DecodeAgentValue(bytes []byte) any {
	fields := DecodeMessage(bytes)

	if fields.Has(PbValueNull) {
		return nil
	}
	if fields.Has(PbValueBool) {
		f := fields.Get(PbValueBool)[0]
		return f.Varint != 0
	}
	if fields.Has(PbValueNumber) {
		b := fields.Get(PbValueNumber)[0].Value
		if len(b) >= 8 {
			bits := binary.LittleEndian.Uint64(b)
			return math.Float64frombits(bits)
		}
		return 0.0
	}
	if fields.Has(PbValueString) {
		return string(fields.Get(PbValueString)[0].Value)
	}
	if fields.Has(PbValueStruct) {
		result := make(map[string]any)
		structMsg := DecodeMessage(fields.Get(PbValueStruct)[0].Value)
		for _, entry := range structMsg.Get(1) { // 1 = fields
			pair := DecodeMessage(entry.Value)
			if pair.Has(1) { // 1 = key
				key := string(pair.Get(1)[0].Value)
				var val any
				if pair.Has(2) { // 2 = value
					val = DecodeAgentValue(pair.Get(2)[0].Value)
				}
				result[key] = val
			}
		}
		return result
	}
	if fields.Has(PbValueList) {
		var list []any
		listMsg := DecodeMessage(fields.Get(PbValueList)[0].Value)
		for _, item := range listMsg.Get(1) { // 1 = values
			list = append(list, DecodeAgentValue(item.Value))
		}
		return list
	}

	return nil
}

// EncodeMcpToolDefinition encodes agent.v1.McpToolDefinition body.
func EncodeMcpToolDefinition(name, description string, schema any) []byte {
	return ConcatBuffers(
		EncodeField(1, WireBytes, name),
		EncodeField(2, WireBytes, description),
		EncodeField(3, WireBytes, EncodeAgentValue(schema)),
		EncodeField(4, WireBytes, "9router"),
		EncodeField(5, WireBytes, name),
	)
}

// EncodeMcpTools encodes multiple tool definitions for AgentRunRequest.mcp_tools.
func EncodeMcpTools(tools []any) []byte {
	if len(tools) == 0 {
		return nil
	}
	var defs [][]byte
	for _, t := range tools {
		tMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		name := ""
		description := ""
		var schema any

		if fn, hasFn := tMap["function"].(map[string]any); hasFn {
			if n, ok := fn["name"].(string); ok {
				name = n
			}
			if d, ok := fn["description"].(string); ok {
				description = d
			}
			schema = fn["parameters"]
		} else {
			if n, ok := tMap["name"].(string); ok {
				name = n
			}
			if d, ok := tMap["description"].(string); ok {
				description = d
			}
			if s, hasS := tMap["parameters"]; hasS {
				schema = s
			} else if s, hasIn := tMap["inputSchema"]; hasIn {
				schema = s
			} else if s, hasIn2 := tMap["input_schema"]; hasIn2 {
				schema = s
			}
		}
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		def := EncodeMcpToolDefinition(name, description, schema)
		defs = append(defs, EncodeField(1, WireBytes, def))
	}
	return ConcatBuffers(defs...)
}

// McpArgs holds decoded MCP tool invocation parameters.
type McpArgs struct {
	Name       string
	ToolName   string
	ToolCallID string
	Args       map[string]any
}

// DecodeMcpArgs parses an incoming MCP execution request.
func DecodeMcpArgs(bytes []byte) McpArgs {
	msg := DecodeMessage(bytes)
	args := make(map[string]any)

	for _, entry := range msg.Get(2) { // 2 = repeated args entry
		pair := DecodeMessage(entry.Value)
		if pair.Has(1) {
			key := string(pair.Get(1)[0].Value)
			if pair.Has(2) {
				args[key] = DecodeAgentValue(pair.Get(2)[0].Value)
			}
		}
	}

	res := McpArgs{Args: args}
	if msg.Has(1) {
		res.Name = string(msg.Get(1)[0].Value)
	}
	if msg.Has(3) {
		res.ToolCallID = string(msg.Get(3)[0].Value)
	}
	if msg.Has(5) {
		res.ToolName = string(msg.Get(5)[0].Value)
	}
	return res
}

// EncodeMcpResultSuccess encodes success payload for an MCP call.
func EncodeMcpResultSuccess(textItems []string, isError bool) []byte {
	var items [][]byte
	for _, text := range textItems {
		textContent := EncodeField(1, WireBytes, EncodeField(1, WireBytes, text))
		items = append(items, EncodeField(1, WireBytes, textContent))
	}
	errVal := 0
	if isError {
		errVal = 1
	}
	items = append(items, EncodeField(2, WireVarint, errVal))
	success := ConcatBuffers(items...)
	return EncodeField(1, WireBytes, success)
}

// EncodeMcpResultError encodes error payload for an MCP call.
func EncodeMcpResultError(message string) []byte {
	errPayload := EncodeField(1, WireBytes, message)
	return EncodeField(2, WireBytes, errPayload)
}

// EncodeMcpResultToolNotFound encodes tool not found payload.
func EncodeMcpResultToolNotFound(name string) []byte {
	tnfPayload := EncodeField(1, WireBytes, name)
	return EncodeField(5, WireBytes, tnfPayload)
}
