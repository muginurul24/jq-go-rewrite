package security

import "testing"

func TestStringCipherRoundTrip(t *testing.T) {
	cipher := NewStringCipher("0123456789abcdef0123456789abcdef")

	encrypted, err := cipher.EncryptString("justqiu-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if encrypted == "justqiu-secret" {
		t.Fatalf("expected ciphertext to differ from plaintext")
	}

	decrypted, err := cipher.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if decrypted != "justqiu-secret" {
		t.Fatalf("expected decrypted plaintext to match, got %q", decrypted)
	}
}

func TestStringCipherRejectsForeignPayload(t *testing.T) {
	first := NewStringCipher("0123456789abcdef0123456789abcdef")
	second := NewStringCipher("fedcba9876543210fedcba9876543210")

	encrypted, err := first.EncryptString("justqiu-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if _, err := second.DecryptString(encrypted); err == nil {
		t.Fatalf("expected decrypt to fail with different key")
	}
}
