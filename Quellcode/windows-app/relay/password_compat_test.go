package main

import (
	"encoding/hex"
	"testing"
)

func TestPasswordHashPBKDF2Compatibility(t *testing.T) {
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}
	result := hex.EncodeToString(passwordHash("example-password-for-test", salt))
	const expected = "d5815745f9bc10f28dedbb449da8bebe7ee8d75e19c24f811a696db3a847a76c"
	if result != expected {
		t.Fatalf("hash format changed: got %s", result)
	}
}
