package tokoz

import (
	"errors"
	"testing"
	"time"
)

func TestConfigureRejectsWeakConfig(t *testing.T) {
	if err := Configure(Config{Secret: "short"}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Configure() error = %v", err)
	}
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	err := Configure(Config{
		Secret: "01234567890123456789012345678901", Issuer: "daftar-test",
		Audience: "daftar-web-test", AccessTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	token, err := GenerateToken("user-01TEST", "user", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ValidateAccessToken(token)
	if err != nil || claims.Subject != "user-01TEST" || claims.Email != "user@example.com" {
		t.Fatalf("ValidateAccessToken() = %#v, %v", claims, err)
	}
}
