package cfg

import (
	"testing"
	"time"
)

func TestLoadEnvironment(t *testing.T) {
	t.Setenv("DAFTAR_JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("DAFTAR_HTTP_ADDRESS", ":9090")
	t.Setenv("DAFTAR_MONGODB_DATABASE", "daftar_test")
	t.Setenv("DAFTAR_CORS_ALLOWED_ORIGINS", "https://one.example.com,https://two.example.com")
	t.Setenv("DAFTAR_JWT_ACCESS_TTL", "8h")

	config, err := Load(t.TempDir() + "/missing.env")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.ServiceName != "daftar-api" || config.HTTP.Address != ":9090" || config.MongoDB.Database != "daftar_test" {
		t.Fatalf("Config = %#v", config)
	}
	if config.JWT.AccessTTL != 8*time.Hour || len(config.HTTP.CORSAllowedOrigins) != 2 {
		t.Fatalf("Config = %#v", config)
	}
}

func TestLoadRejectsMissingJWTSecret(t *testing.T) {
	t.Setenv("DAFTAR_JWT_SECRET", "")
	if _, err := Load(t.TempDir() + "/missing.env"); err == nil {
		t.Fatal("Load() error = nil")
	}
}
