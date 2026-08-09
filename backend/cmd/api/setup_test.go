package main

import (
	"context"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/cfg"
)

func TestSetupRejectsWeakJWTBeforeConnecting(t *testing.T) {
	config := cfg.Default()
	config.Environment = cfg.EnvironmentProduction
	config.Cookie.Secure = true
	config.JWT.Secret = "weak"

	application, err := setup(context.Background(), config)
	if err == nil || application != nil {
		t.Fatalf("setup() = %#v, %v", application, err)
	}
}
