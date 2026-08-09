package requestctx

import (
	"context"
	"testing"

	"github.com/babafemi99/daftar/backend/internal/model"
)

func TestRequestValues(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-test")
	ctx = WithExecutor(ctx, model.Executor{ID: "user-test", Email: "user@example.com", Role: "user"})

	if requestID, ok := RequestID(ctx); !ok || requestID != "req-test" {
		t.Fatalf("RequestID() = %q, %v", requestID, ok)
	}
	executor, ok := Executor(ctx)
	if !ok || executor.ID != "user-test" {
		t.Fatalf("Executor() = %#v, %v", executor, ok)
	}
}

func TestMissingRequestValues(t *testing.T) {
	if _, ok := RequestID(context.Background()); ok {
		t.Fatal("RequestID() ok = true")
	}
	if _, ok := Executor(context.Background()); ok {
		t.Fatal("Executor() ok = true")
	}
}
