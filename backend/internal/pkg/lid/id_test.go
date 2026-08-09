package lid

import (
	"errors"
	"strings"
	"testing"
)

func TestResourceIDs(t *testing.T) {
	tests := []struct {
		name   string
		prefix Prefix
		newID  func() string
	}{
		{name: "user", prefix: User, newID: NewUser},
		{name: "document", prefix: Document, newID: NewDocument},
		{name: "line", prefix: Line, newID: NewLine},
		{name: "request", prefix: Request, newID: NewRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.newID()
			if !strings.HasPrefix(id, string(tt.prefix)+separator) {
				t.Fatalf("ID = %q, want %q prefix", id, tt.prefix)
			}
			if err := Validate(id, tt.prefix); err != nil {
				t.Fatalf("Validate(%q) error = %v", id, err)
			}
		})
	}
}

func TestIDsAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1_000)
	for range 1_000 {
		id := NewDocument()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestValidateRejectsWrongPrefix(t *testing.T) {
	err := Validate(NewUser(), Document)
	if !errors.Is(err, ErrInvalidPrefix) {
		t.Fatalf("Validate() error = %v, want ErrInvalidPrefix", err)
	}
}

func TestValidateRejectsInvalidULID(t *testing.T) {
	err := Validate("user-not-a-ulid", User)
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Validate() error = %v, want ErrInvalidID", err)
	}
}
