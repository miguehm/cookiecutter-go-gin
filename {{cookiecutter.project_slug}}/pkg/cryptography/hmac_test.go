package cryptography

import "testing"

func TestGenerateHMACKnownVector(t *testing.T) {
	got := GenerateHMAC([]byte("message"), "secret")
	want := "8b5f48702995c1598c573db1e21866a9b825d4a794d169d7060a03605796360b"
	if got != want {
		t.Fatalf("GenerateHMAC = %q, want %q", got, want)
	}
}

func TestGenerateHMACIsDeterministicAndDistinct(t *testing.T) {
	first := GenerateHMAC([]byte("message"), "secret")
	if again := GenerateHMAC([]byte("message"), "secret"); again != first {
		t.Fatalf("GenerateHMAC is not deterministic: %q vs %q", again, first)
	}
	if other := GenerateHMAC([]byte("other"), "secret"); other == first {
		t.Fatal("different messages must not collide")
	}
	if other := GenerateHMAC([]byte("message"), "other"); other == first {
		t.Fatal("different secrets must not collide")
	}
}
