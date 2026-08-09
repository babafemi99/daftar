package mercury

import (
	"context"
	"errors"
	"testing"
)

func TestNewCoreRequiresDatabase(t *testing.T) {
	core, err := NewCore(nil)
	if !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("NewCore(nil) error = %v, want ErrDatabaseRequired", err)
	}
	if core != nil {
		t.Fatalf("NewCore(nil) = %#v, want nil", core)
	}
}

func TestNilCoreCannotEnsureIndexes(t *testing.T) {
	var core *Core
	if err := core.EnsureIndexes(context.Background()); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("EnsureIndexes() error = %v, want ErrDatabaseRequired", err)
	}
}

func TestNilCoreCannotEnsureSchemaValidators(t *testing.T) {
	var core *Core
	if err := core.EnsureSchemaValidators(context.Background()); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("EnsureSchemaValidators() error = %v, want ErrDatabaseRequired", err)
	}
}

func TestConnectRequiresConfiguration(t *testing.T) {
	ctx := context.Background()

	if _, err := Connect(ctx, "", "daftar"); !errors.Is(err, ErrMongoURIRequired) {
		t.Fatalf("Connect() missing URI error = %v, want ErrMongoURIRequired", err)
	}
	if _, err := Connect(ctx, "mongodb://localhost:27017", ""); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("Connect() missing database error = %v, want ErrDatabaseRequired", err)
	}
}

func TestNilCoreTransactionAndShutdown(t *testing.T) {
	var core *Core
	if err := core.RunInTx(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("RunInTx() error = %v, want ErrDatabaseRequired", err)
	}
	if err := core.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
