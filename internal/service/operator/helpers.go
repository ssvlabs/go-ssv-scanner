package operator

import (
	"bytes"
	"encoding/base64"
	"math/big"
)

// normalizeDynamicBytes attempts to unwrap ABI dynamic-bytes encoding if detected.
// If the byte slice looks like: [32-byte head=0x20][32-byte length=N][N bytes data][padding],
// it returns the N bytes of data. Otherwise, returns the input unchanged.
func normalizeDynamicBytes(b []byte) []byte {
	if len(b) < 64 {
		return b
	}
	// big-endian uint256 read helper
	readUint256 := func(word []byte) int64 {
		if len(word) != 32 {
			return -1
		}
		var n big.Int
		n.SetBytes(word)
		return n.Int64()
	}
	head := readUint256(b[0:32])
	if head != 32 { // first head pointing to start of tail
		return b
	}
	length := readUint256(b[32:64])
	if length < 0 {
		return b
	}
	start := 64
	end := start + int(length)
	if end > len(b) || start > len(b) {
		return b
	}
	return b[start:end]
}

// isBase64PEM returns true if b decodes from base64 to a PEM-like header starting with "-----BEGIN"
func isBase64PEM(b []byte) bool {
	// quick reject if not printable-ish; still attempt decode
	dec, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		return false
	}
	return bytes.HasPrefix(dec, []byte("-----BEGIN"))
}
