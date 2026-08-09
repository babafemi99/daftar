package lid

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
)

const separator = "-"

type Prefix string

const (
	User     Prefix = "user"
	Document Prefix = "doc"
	Line     Prefix = "line"
	Request  Prefix = "req"
	Audit    Prefix = "audit"
	Session  Prefix = "session"
)

var (
	ErrInvalidID     = errors.New("invalid identifier")
	ErrInvalidPrefix = errors.New("invalid identifier prefix")
)

func NewUser() string     { return newID(User) }
func NewDocument() string { return newID(Document) }
func NewLine() string     { return newID(Line) }
func NewRequest() string  { return newID(Request) }
func NewAudit() string    { return newID(Audit) }
func NewSession() string  { return newID(Session) }

func newID(prefix Prefix) string {
	return string(prefix) + separator + ulid.Make().String()
}

// Validate checks both the resource prefix and canonical ULID representation.
func Validate(value string, expected Prefix) error {
	prefix, rawULID, found := strings.Cut(value, separator)
	if !found || Prefix(prefix) != expected {
		return fmt.Errorf("%w: expected %s prefix", ErrInvalidPrefix, expected)
	}
	if strings.Contains(rawULID, separator) {
		return fmt.Errorf("%w: unexpected separator", ErrInvalidID)
	}
	if _, err := ulid.ParseStrict(rawULID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidID, err)
	}

	return nil
}
