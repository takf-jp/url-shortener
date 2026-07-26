package auth

import "testing"

func TestHashPassword_VerifyPassword_一致(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if err := VerifyPassword(hash, "password123"); err != nil {
		t.Errorf("expected password to match, got %v", err)
	}
}

func TestVerifyPassword_違うパスワードで失敗(t *testing.T) {
	hash, _ := HashPassword("password123")
	if err := VerifyPassword(hash, "wrongpassword"); err == nil {
		t.Error("expected wrong password, got nil error")
	}
}

func TestHashPassword_同じパスワードで毎回違うハッシュ(t *testing.T) {
	hash1, _ := HashPassword("password123")
	hash2, _ := HashPassword("password123")

	if hash1 == hash2 {
		t.Error("expected different hashes for the same password, got the same")
	}
}
