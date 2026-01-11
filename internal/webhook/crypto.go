package webhook

import "crypto/rand"

// cryptoRandReadReal uses crypto/rand for secure random generation.
func cryptoRandReadReal(b []byte) (int, error) {
	return rand.Read(b)
}
