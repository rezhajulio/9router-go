package cursor

import (
	"encoding/binary"
	"fmt"
	"io"
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
	FieldRunRequest        = 1
	FieldAction            = 2
	FieldModelDetails      = 3
	FieldMcpTools          = 4
	FieldRequestedModel    = 9
	FieldInteractionUpdate = 1
	FieldExecRequest       = 2
	FieldKvServerMessage   = 4
	FieldTextDelta         = 1
	FieldThinkingDelta     = 4
	FieldTurnEnded         = 14
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
		// A length that doesn't fit in a (signed, 64-bit-safe) int would turn
		// end := nextPos + int(length) negative, passing the "end > len"
		// bounds check and panicking on the slice below. Reject it before
		// converting instead of trusting an attacker-controlled varint.
		if length > uint64(len(buffer)-nextPos) {
			return 0, 0, DecodedField{}, offset, fmt.Errorf("length delimiter exceeds buffer")
		}
		end := nextPos + int(length)
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
	fields, _ := DecodeMessageStrict(data)
	return fields
}

// DecodeMessageStrict decodes a protobuf message and reports a decode error
// instead of returning the fields a truncated payload happened to contain.
// Callers that turn the payload into client-visible data (a tool call's
// arguments) must use this so a partial decode is refused rather than sent.
func DecodeMessageStrict(data []byte) (DecodedMessage, error) {
	fields := make(DecodedMessage)
	pos := 0

	for pos < len(data) {
		fNum, _, field, nextPos, err := DecodeField(data, pos)
		if err != nil {
			return fields, err
		}
		fields[fNum] = append(fields[fNum], field)
		pos = nextPos
	}

	return fields, nil
}
