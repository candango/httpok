package security

import (
	"math/rand"
	"time"
)

// RandomString generates a random string of length 's' composed of
// lowercase letters, uppercase letters, and digits.
func RandomString(s int) string {
	asciiLower := "abcdefghijklmnopqrstuvwxyz"
	asciiUpper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "012345679"
	chars := []rune(asciiLower + asciiUpper + digits)
	rand.NewSource(time.Now().UnixNano())
	r := make([]rune, s)
	for i := range r {
		r[i] = chars[rand.Intn(len(chars))]
	}
	return string(r)
}
