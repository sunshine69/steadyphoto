package database

import (
	"testing"
)

func TestPostgresSessionRepository_Hashing(t *testing.T) {
    repo := &PostgresSessionRepository{} // Manual init since DB isn't connected here
    token := "test-opaque-token"
    hash1 := repo.HashToken(token)
    hash2 := repo.HashToken(token)

    if hash1 != hash2 {
        t.Errorf("Hashing is not deterministic: %s vs %s", hash1, hash2)
    }
}
