package cursor

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

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

const maxAgentValueDepth = 100

// DecodeAgentValue decodes google.protobuf.Value bytes to Go any.
func DecodeAgentValue(bytes []byte) any {
	return decodeAgentValueDepth(bytes, 0)
}

func decodeAgentValueDepth(bytes []byte, depth int) any {
	if depth > maxAgentValueDepth {
		return nil
	}

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
					val = decodeAgentValueDepth(pair.Get(2)[0].Value, depth+1)
				}
				result[key] = val
			}
		}
		return result
	}
	if fields.Has(PbValueList) {
		list := make([]any, 0)
		listMsg := DecodeMessage(fields.Get(PbValueList)[0].Value)
		for _, item := range listMsg.Get(1) { // 1 = values
			list = append(list, decodeAgentValueDepth(item.Value, depth+1))
		}
		return list
	}

	return nil
}
