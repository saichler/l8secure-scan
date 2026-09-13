package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// ComputeHMACSHA256 computes the HMAC-SHA256 of payload using secret,
// returning the digest as a lowercase hex string.
func ComputeHMACSHA256(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMACSHA256Hex reports whether signatureHex (a hex-encoded digest, no
// prefix) matches the HMAC-SHA256 of payload computed with secret. Returns
// false if signatureHex is not valid hex or the HMAC does not match.
func VerifyHMACSHA256Hex(payload []byte, signatureHex, secret string) bool {
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}
