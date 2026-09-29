package cursor

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"math"
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
