package service

import (
	"strings"
	"testing"
)

func TestGenerateCode_長さが指定通り(t *testing.T) {
	code, err := generateCode(6)
	if err != nil {
		t.Fatalf("generateCode failed: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected length 6, got %d", len(code))
	}
}

func TestGenerateCode_文字セット範囲内(t *testing.T) {
	code, err := generateCode(100) //多めに生成
	if err != nil {
		t.Fatalf("generateCode failed: %v", err)
	}
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("character %c is not in charset", c)
		}
	}
}

