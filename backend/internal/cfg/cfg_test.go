package cfg

import (
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	config := Default()

	if config.Environment != EnvironmentDevelopment {
		t.Errorf("Environment = %q", config.Environment)
	}
	if config.HTTP.Address != ":8080" || config.HTTP.MaxHeaderBytes != 1<<20 {
		t.Errorf("HTTP = %#v", config.HTTP)
	}
	if config.MongoDB.ConnectTimeout <= 0 || config.JWT.AccessTTL <= 0 {
		t.Fatalf("Config = %#v", config)
	}
}

func TestHTTPWithDefaultsPreservesExplicitValues(t *testing.T) {
	config := (HTTP{Address: ":9000", ReadTimeout: time.Minute, MaxHeaderBytes: 2048}).WithDefaults()

	if config.Address != ":9000" || config.ReadTimeout != time.Minute || config.MaxHeaderBytes != 2048 {
		t.Fatalf("WithDefaults() = %#v", config)
	}
	if config.WriteTimeout <= 0 || config.ShutdownTimeout <= 0 {
		t.Fatalf("WithDefaults() did not fill omitted values: %#v", config)
	}
}

func TestProductionRejectsWeakJWTSecret(t *testing.T) {
	config := Default()
	config.Environment = EnvironmentProduction
	config.Cookie.Secure = true
	config.JWT.Secret = "short"
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestCookiePolicyValidation(t *testing.T) {
	config := Default()
	config.JWT.Secret = "01234567890123456789012345678901"
	config.Cookie.SameSite = "invalid"
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() invalid SameSite error = nil")
	}

	config.Cookie.SameSite = "none"
	config.Cookie.Secure = false
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() insecure SameSite=None error = nil")
	}
}

func TestBootstrapUserValidation(t *testing.T) {
	config := Default()
	config.JWT.Secret = "01234567890123456789012345678901"
	config.BootstrapUser.Enabled = true
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() incomplete bootstrap user error = nil")
	}
}

func TestLoggingValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "unsupported level", mutate: func(config *Config) { config.Logging.Level = "trace" }},
		{name: "unsupported format", mutate: func(config *Config) { config.Logging.Format = "xml" }},
		{name: "invalid slow threshold", mutate: func(config *Config) { config.Logging.SlowRequestThreshold = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Default()
			config.JWT.Secret = "01234567890123456789012345678901"
			test.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
