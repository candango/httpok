package util

import "math/rand"

func RandomString(s int) string {
	asciiLower := "abcdefghijklmnopqrstuvwxyz"
	asciiUpper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "012345679"
	chars := []rune(asciiLower + asciiUpper + digits)
	r := make([]rune, s)
	for i := range r {
		r[i] = chars[rand.Intn(len(chars))]
	}
	return string(r)
}
