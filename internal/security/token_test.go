package security

import (
	"testing"
)

func TestGenerateRandomToken(t *testing.T) {
	length := 32
	expectedHexLen := length * 2

	token1, err := GenerateRandomToken(length)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 1. Test Length
	if len(token1) != expectedHexLen {
		t.Errorf("Expected hex length %d, got %d", expectedHexLen, len(token1))
	}

	// 2. Test Uniqueness (Collision Check)
	token2, err := GenerateRandomToken(length)
	if err != nil {
		t.Fatalf("Failed to generate second token: %v", err)
	}

	if token1 == token2 {
		t.Error("Generated two identical tokens; collision detected!")
	}

	// 3. Test Entropy (Very basic check for non-obvious pattern)
	// We don't do full statistical tests here, but ensure it's not just zeros/ones
	allZeros := true
	for _, char := range token1 {
		if char != '0' {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Generated token appears to be only zeros")
	}
}
