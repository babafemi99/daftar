package buhari

const validationMessage = "One or more fields are invalid."

// FieldError identifies a validation failure at a specific input path.
type FieldError struct {
	Path    string `json:"path"`
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func Validation(fields ...FieldError) *Error {
	return &Error{
		Code:    CodeValidationFailed,
		Message: validationMessage,
		Fields:  fields,
	}
}

func InvalidFilter(fields ...FieldError) *Error {
	return &Error{
		Code:    CodeInvalidFilter,
		Message: "One or more filters are invalid.",
		Fields:  fields,
	}
}
