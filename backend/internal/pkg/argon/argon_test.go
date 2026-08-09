package argon

import (
	"errors"
	"io"
	"strings"
	"testing"
)

var testParams = Params{
	Memory:      8 * 1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func TestHashAndVerify(t *testing.T) {
	encoded, err := hash("correct horse battery staple", testParams, strings.NewReader("0123456789abcdef"))
	if err != nil {
		t.Fatalf("hash() error = %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Fatalf("hash = %q", encoded)
	}

	matched, err := Verify("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !matched {
		t.Fatal("Verify() = false, want true")
	}

	matched, err = Verify("wrong password", encoded)
	if err != nil {
		t.Fatalf("Verify() wrong password error = %v", err)
	}
	if matched {
		t.Fatal("Verify() wrong password = true, want false")
	}
}

func TestHashUsesUniqueSalt(t *testing.T) {
	first, err := Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() first error = %v", err)
	}
	second, err := Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() second error = %v", err)
	}
	if first == second {
		t.Fatal("Hash() produced identical hashes for independently salted passwords")
	}
}

func TestVerifyRejectsInvalidHashes(t *testing.T) {
	tests := []string{
		"",
		"not-a-hash",
		"$argon2i$v=19$m=8192,t=1,p=1$c2FsdHNhbHRzYWx0$YWJjZGVmZ2hpamtsbW5vcA",
		"$argon2id$v=18$m=8192,t=1,p=1$c2FsdHNhbHRzYWx0$YWJjZGVmZ2hpamtsbW5vcA",
		"$argon2id$v=19$m=999999,t=1,p=1$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY",
	}

	for _, encoded := range tests {
		_, err := Verify("password", encoded)
		if !errors.Is(err, ErrInvalidHash) {
			t.Errorf("Verify(%q) error = %v, want ErrInvalidHash", encoded, err)
		}
	}
}

func TestHashReportsRandomnessFailure(t *testing.T) {
	_, err := hash("password", testParams, failingReader{})
	if err == nil {
		t.Fatal("hash() error = nil")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
