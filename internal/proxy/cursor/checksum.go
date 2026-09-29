package cursor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var cursorTimezone = sync.OnceValue(func() string {
	if tz := strings.TrimPrefix(os.Getenv("TZ"), ":"); tz != "" {
		return tz
	}
	if p, err := filepath.EvalSymlinks("/etc/localtime"); err == nil {
		if _, name, ok := strings.Cut(p, "zoneinfo/"); ok {
			return name
		}
	}
	return "UTC"
})

// GenerateHashed64Hex returns SHA-256 hex digest of input + salt.
func GenerateHashed64Hex(input, salt string) string {
	h := sha256.Sum256([]byte(input + salt))
	return hex.EncodeToString(h[:])
}

// GenerateSessionID returns UUID v5 with DNS namespace from authToken.
func GenerateSessionID(authToken string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(authToken)).String()
}

// GenerateCursorChecksum generates the Jyh cipher checksum header.
// Format: {base64_url_encoded}{machineId}
func GenerateCursorChecksum(machineID string) string {
	return cursorChecksumAt(time.Now().UnixMilli()/1000000, machineID)
}

func cursorChecksumAt(timestamp int64, machineID string) string {
	// JavaScript bitwise operators convert operands to 32-bit signed integers
	// and mask shift amounts to 5 bits (shift mod 32), so (ts >> 40) is
	// (int32(ts) >> 8) and (ts >> 32) is (int32(ts) >> 0).
	ts := int32(timestamp)
	byteArray := []byte{
		byte((ts >> 8) & 0xFF),
		byte(ts & 0xFF),
		byte((ts >> 24) & 0xFF),
		byte((ts >> 16) & 0xFF),
		byte((ts >> 8) & 0xFF),
		byte(ts & 0xFF),
	}

	var t byte = 165
	for i := 0; i < len(byteArray); i++ {
		byteArray[i] = ((byteArray[i] ^ t) + byte(i%256)) & 0xFF
		t = byteArray[i]
	}

	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var encoded []byte

	for i := 0; i < len(byteArray); i += 3 {
		a := byteArray[i]
		var b, c byte
		if i+1 < len(byteArray) {
			b = byteArray[i+1]
		}
		if i+2 < len(byteArray) {
			c = byteArray[i+2]
		}

		encoded = append(encoded, alphabet[a>>2])
		encoded = append(encoded, alphabet[((a&3)<<4)|(b>>4)])

		if i+1 < len(byteArray) {
			encoded = append(encoded, alphabet[((b&15)<<2)|(c>>6)])
		}
		if i+2 < len(byteArray) {
			encoded = append(encoded, alphabet[c&63])
		}
	}

	return string(encoded) + machineID
}

// BuildCursorHeaders creates all required Cursor API HTTP headers.
func BuildCursorHeaders(accessToken, machineID string, ghostMode bool) map[string]string {
	cleanToken := accessToken
	if _, after, ok := strings.Cut(accessToken, "::"); ok {
		cleanToken, _, _ = strings.Cut(after, "::")
	}

	effectiveMachineID := machineID
	if effectiveMachineID == "" {
		effectiveMachineID = GenerateHashed64Hex(cleanToken, "machineId")
	}

	sessionID := GenerateSessionID(cleanToken)
	clientKey := GenerateHashed64Hex(cleanToken, "")
	checksum := GenerateCursorChecksum(effectiveMachineID)

	os := "linux"
	if runtime.GOOS == "windows" {
		os = "windows"
	} else if runtime.GOOS == "darwin" {
		os = "macos"
	}

	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "aarch64"
	}

	ghostVal := "true"
	if !ghostMode {
		ghostVal = "false"
	}

	tz := cursorTimezone()

	return map[string]string{
		"authorization":               "Bearer " + cleanToken,
		"connect-accept-encoding":     "gzip",
		"connect-protocol-version":    "1",
		"content-type":                "application/connect+proto",
		"user-agent":                  "connect-es/1.6.1",
		"x-amzn-trace-id":             fmt.Sprintf("Root=%s", uuid.New().String()),
		"x-client-key":                clientKey,
		"x-cursor-checksum":           checksum,
		"x-cursor-client-version":     "3.12.17",
		"x-cursor-client-commit":      "0fb762053c34788bb7760d5673f8a6d4c8589d50",
		"x-cursor-client-type":        "ide",
		"x-cursor-client-os":          os,
		"x-cursor-client-arch":        arch,
		"x-cursor-client-device-type": "desktop",
		"x-cursor-config-version":     uuid.New().String(),
		"x-cursor-timezone":           tz,
		"x-ghost-mode":                ghostVal,
		"x-request-id":                uuid.New().String(),
		"x-session-id":                sessionID,
	}
}
