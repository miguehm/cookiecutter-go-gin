package cryptography

import (
	"strings"
	"testing"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hashed, err := HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if string(hashed) == "s3cret-password" {
		t.Fatal("HashPassword returned the plain password")
	}
	if err := VerifyPassword(string(hashed), "s3cret-password"); err != nil {
		t.Fatalf("VerifyPassword rejected the correct password: %v", err)
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hashed, err := HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if err := VerifyPassword(string(hashed), "wrong"); err == nil {
		t.Fatal("VerifyPassword accepted the wrong password")
	}
}

func TestHashSHA256(t *testing.T) {
	got := HashSHA256([]byte("abc"))

	if got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("HashSHA256 = %q", got)
	}
	if !isHex(got) {
		t.Fatalf("HashSHA256 is not hex encoded: %q", got)
	}
}

func TestHashSHA256IsDeterministicAndDistinct(t *testing.T) {
	first := HashSHA256([]byte("a"))
	second := HashSHA256([]byte("a"))
	other := HashSHA256([]byte("b"))

	if first != second {
		t.Fatal("HashSHA256 is not deterministic")
	}
	if first == other {
		t.Fatal("HashSHA256 collided on different inputs")
	}
}

func isHex(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return len(s) == 64
}
