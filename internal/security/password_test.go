package security

import (
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "mySecretPassword123!"
	
	// 1. Test Hashing
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("Hashed password is empty")
	}

	// 2. Test Correct Password
	match := CheckPasswordHash(password, hash)
	if !match {
		t.Error("CheckPasswordHash failed to match the correct password")
	}

	// 3. Test Incorrect Password
	wrongPassword := "notTheSecret"
	noMatch := CheckPasswordHash(wrongPassword, hash)
	if noMatch {
		t.Error("CheckPasswordHash matched an incorrect password")
	}

	// 4. Test Empty Password
	emptyHash, err := HashPassword("")
	if err != nil {
		t.Fatalf("Failed to hash empty password: %v", err)
	}
	if !CheckPasswordHash("", emptyHash) {
		t.Error("CheckPasswordHash failed on empty string")
	}
}
