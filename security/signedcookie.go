// Package security contains security-related helpers used by httpok.
package security

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // required for Tornado v1 compatibility
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"strconv"
	"strings"
	"time"
)

const (
	signedCookieV2  = "2"
	maxCookieFuture = 31 * 24 * time.Hour
)

// CreateSignedValue creates a version 2 signed value using key version 0.
//
// It preserves the original API for applications that use one cookie secret.
// Use CreateSignedValueWithKeyVersion when key rotation is configured.
func CreateSignedValue(
	secret []byte,
	name, value string,
	now time.Time,
) string {
	return CreateSignedValueWithKeyVersion(secret, 0, name, value, now)
}

// CreateSignedValueWithKeyVersion creates a Tornado-compatible version 2
// signed value using keyVersion.
//
// The value contains the session value encoded with standard base64 and is
// authenticated with HMAC-SHA256. The timestamp is taken from now, which also
// makes the function deterministic for callers that need test vectors.
func CreateSignedValueWithKeyVersion(
	secret []byte,
	keyVersion int,
	name, value string,
	now time.Time,
) string {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	encodedValue := base64.StdEncoding.EncodeToString([]byte(value))
	toSign := strings.Join([]string{
		signedCookieV2,
		formatTornadoField(strconv.Itoa(keyVersion)),
		formatTornadoField(timestamp),
		formatTornadoField(name),
		formatTornadoField(encodedValue),
		"",
	}, "|")

	signature := tornadoHMAC(sha256.New, secret, toSign)
	return toSign + hex.EncodeToString(signature)
}

// DecodeSignedValue verifies a signed value using key version 0.
//
// It preserves the original API for applications that use one cookie secret.
// Use DecodeSignedValueWithKeys when key rotation is configured.
func DecodeSignedValue(
	secret []byte,
	name, signedValue string,
	maxAge time.Duration,
	now time.Time,
) (string, bool) {
	return DecodeSignedValueWithKeys(
		map[int][]byte{0: secret}, name, signedValue, maxAge, now,
	)
}

// DecodeSignedValueWithKeys verifies a signed value against configured keys.
// Version 2 values select their key by the embedded key version. Legacy
// version 1 values are checked against every configured key because v1 has no
// key-version field.
func DecodeSignedValueWithKeys(
	keys map[int][]byte,
	name, signedValue string,
	maxAge time.Duration,
	now time.Time,
) (string, bool) {
	if len(keys) == 0 || signedValue == "" {
		return "", false
	}

	switch signedValueVersion(signedValue) {
	case 1:
		for _, secret := range keys {
			if value, ok := decodeTornadoV1(secret, name, signedValue, maxAge, now); ok {
				return value, true
			}
		}
	case 2:
		return decodeTornadoV2(keys, name, signedValue, maxAge, now)
	}
	return "", false
}

func signedValueVersion(value string) int {
	versionField, _, _ := strings.Cut(value, "|")
	version, err := strconv.Atoi(versionField)
	if err != nil || version > 999 {
		return 1
	}
	return version
}

func formatTornadoField(value string) string {
	return strconv.Itoa(len(value)) + ":" + value
}

func tornadoHMAC(newHash func() hash.Hash, secret []byte, value string) []byte {
	mac := hmac.New(newHash, secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func decodeTornadoV2(
	keys map[int][]byte,
	name, signedValue string,
	maxAge time.Duration,
	now time.Time,
) (string, bool) {
	parts := strings.Split(signedValue, "|")
	if len(parts) != 6 || parts[0] != signedCookieV2 {
		return "", false
	}

	keyVersionField, ok := parseTornadoField(parts[1])
	if !ok {
		return "", false
	}
	keyVersion, err := strconv.Atoi(keyVersionField)
	if err != nil || keyVersion < 0 {
		return "", false
	}
	secret, ok := keys[keyVersion]
	if !ok || len(secret) == 0 {
		return "", false
	}
	timestamp, ok := parseTornadoField(parts[2])
	if !ok || !validTornadoTimestamp(timestamp, maxAge, now) {
		return "", false
	}
	nameField, ok := parseTornadoField(parts[3])
	if !ok || nameField != name {
		return "", false
	}
	encodedValue, ok := parseTornadoField(parts[4])
	if !ok {
		return "", false
	}
	if len(parts[5]) != sha256.Size*2 {
		return "", false
	}
	passedSignature, err := hex.DecodeString(parts[5])
	if err != nil {
		return "", false
	}

	signed := signedValue[:len(signedValue)-len(parts[5])]
	expectedSignature := tornadoHMAC(sha256.New, secret, signed)
	if !hmac.Equal(passedSignature, expectedSignature) {
		return "", false
	}

	value, err := base64.StdEncoding.DecodeString(encodedValue)
	if err != nil {
		return "", false
	}
	return string(value), true
}

func decodeTornadoV1(
	secret []byte,
	name, signedValue string,
	maxAge time.Duration,
	now time.Time,
) (string, bool) {
	parts := strings.Split(signedValue, "|")
	if len(parts) != 3 {
		return "", false
	}

	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write([]byte(name))
	_, _ = mac.Write([]byte(parts[0]))
	_, _ = mac.Write([]byte(parts[1]))
	if !hmac.Equal([]byte(parts[2]), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return "", false
	}

	if !validTornadoV1Timestamp(parts[1], maxAge, now) {
		return "", false
	}
	value, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	return string(value), true
}

func parseTornadoField(value string) (string, bool) {
	separator := strings.IndexByte(value, ':')
	if separator <= 0 {
		return "", false
	}
	length, err := strconv.Atoi(value[:separator])
	if err != nil || length < 0 {
		return "", false
	}
	field := value[separator+1:]
	if len(field) != length {
		return "", false
	}
	return field, true
}

func validTornadoTimestamp(value string, maxAge time.Duration, now time.Time) bool {
	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil || timestamp < 0 {
		return false
	}
	return !time.Unix(timestamp, 0).Before(now.Add(-maxAge))
}

func validTornadoV1Timestamp(value string, maxAge time.Duration, now time.Time) bool {
	if strings.HasPrefix(value, "0") || !validTornadoTimestamp(value, maxAge, now) {
		return false
	}

	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false
	}
	return !time.Unix(timestamp, 0).After(now.Add(maxCookieFuture))
}
