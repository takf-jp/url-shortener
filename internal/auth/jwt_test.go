package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 発行 → 検証で User ID が取れる
func TestJWTService_Issue_Verify_正常系(t *testing.T) {
	// 暗号処理は本物を使う
	s := NewJWTService("test-secret", time.Hour)

	token, err := s.Issue(42)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	claims, err := s.Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}
}

func TestJWTService_Verify_期限切れトークン(t *testing.T) {
	s := NewJWTService("test-secret", -time.Hour) // 発行時点で1時間前に失効

	token, err := s.Issue(1)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	if _, err := s.Verify(token); err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTService_Verify_改ざん(t *testing.T) {
	s := NewJWTService("test-secret", time.Hour)

	token, err := s.Issue(1)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	tamperedToken := token + "tampered" // トークンを改ざん,署名部分を壊す
	if _, err := s.Verify(tamperedToken); err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestJWTService_Verify_別アルゴリズムを拒否(t *testing.T) {
	s := NewJWTService("test-secret", time.Hour)

	// HS512 で署名した(=このサービスが想定しないアルゴリズムの)トークンを自作する
	claims := Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signedToken, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	if _, err := s.Verify(signedToken); err == nil {
		t.Fatal("expected error for HS512 token, got nil")
	}
}
