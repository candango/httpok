package security

import (
	"testing"
	"time"
)

func TestCreateSignedValueMatchesTornadoV2Vector(t *testing.T) {
	got := CreateSignedValue(
		[]byte("secret"),
		"sid",
		"session-123",
		time.Unix(1700000000, 0),
	)

	want := "2|1:0|10:1700000000|3:sid|16:c2Vzc2lvbi0xMjM=|c0913863879591acc16bae28252f735d90e064d5f45001585bfdde608b533d7c"
	if got != want {
		t.Fatalf("signed value = %q, want %q", got, want)
	}
}

func TestDecodeSignedValueAcceptsTornadoVectors(t *testing.T) {
	const (
		secret = "secret"
		name   = "sid"
	)

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "v1",
			value: "c2Vzc2lvbi0xMjM=|1700000000|fadd84217813661b81c728504ed1ce561a1a7510",
			want:  "session-123",
		},
		{
			name:  "v2",
			value: "2|1:0|10:1700000000|3:sid|16:c2Vzc2lvbi0xMjM=|c0913863879591acc16bae28252f735d90e064d5f45001585bfdde608b533d7c",
			want:  "session-123",
		},
	}

	now := time.Unix(1700000000, 0).Add(time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DecodeSignedValue(
				[]byte(secret),
				name,
				tt.value,
				24*time.Hour,
				now,
			)
			if !ok {
				t.Fatal("DecodeSignedValue rejected a valid Tornado vector")
			}
			if got != tt.want {
				t.Fatalf("decoded value = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSignedValueKeyRotation(t *testing.T) {
	now := time.Unix(1700000000, 0)
	oldValue := CreateSignedValueWithKeyVersion(
		[]byte("old-secret"), 1, "sid", "session-123", now,
	)
	newValue := CreateSignedValueWithKeyVersion(
		[]byte("new-secret"), 2, "sid", "session-123", now,
	)
	keys := map[int][]byte{
		1: []byte("old-secret"),
		2: []byte("new-secret"),
	}

	for name, signedValue := range map[string]string{
		"old key": oldValue,
		"new key": newValue,
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := DecodeSignedValueWithKeys(
				keys, "sid", signedValue, 24*time.Hour, now.Add(time.Second),
			)
			if !ok {
				t.Fatal("decoder rejected a value signed by a configured key")
			}
			if got != "session-123" {
				t.Fatalf("decoded value = %q, want %q", got, "session-123")
			}
		})
	}

	_, ok := DecodeSignedValueWithKeys(
		map[int][]byte{2: []byte("new-secret")},
		"sid",
		oldValue,
		24*time.Hour,
		now.Add(time.Second),
	)
	if ok {
		t.Fatal("decoder accepted a value signed by an unknown key")
	}
}

func TestDecodeSignedValueRejectsTamperedAndExpiredValues(t *testing.T) {
	const value = "2|1:0|10:1700000000|3:sid|16:c2Vzc2lvbi0xMjM=|c0913863879591acc16bae28252f735d90e064d5f45001585bfdde608b533d7c"

	tests := []struct {
		name   string
		value  string
		maxAge time.Duration
		now    time.Time
	}{
		{
			name:   "tampered signature",
			value:  value[:len(value)-1] + "e",
			maxAge: 24 * time.Hour,
			now:    time.Unix(1700000001, 0),
		},
		{
			name:   "expired timestamp",
			value:  value,
			maxAge: time.Second,
			now:    time.Unix(1700000010, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := DecodeSignedValue(
				[]byte("secret"),
				"sid",
				tt.value,
				tt.maxAge,
				tt.now,
			)
			if ok {
				t.Fatal("DecodeSignedValue accepted an invalid value")
			}
		})
	}
}
