package buhari

// Error describes a safe application failure. Cause is retained for logging
// and error inspection, but must never be serialized to an API response.
type Error struct {
	Code    Code
	Message string
	Fields  []FieldError
	Cause   error
}

func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func Wrap(code Code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	return string(e.Code) + ": " + e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}
