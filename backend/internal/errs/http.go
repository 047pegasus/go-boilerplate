package errs

import "strings"

// For Form field error handling
type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// To define custom actions such as redirects etc.
type ActionType string

const (
	ActionTypeRedirect ActionType = "redirect"
)

type Action struct {
	Type    ActionType `json:"type"`
	Message string     `json:"message"`
	Value   string     `json:"value"`
}

type HttpError struct {
	Code       string       `json:"code"`     // for sending app level codes like TODO_NOT_FOUND etc..
	Message    string       `json:"message"`  //Actual message to be transmitted
	StatusCode int          `json:"status"`   //actual HTTP status codes
	Override   bool         `json:"override"` //mostly unused but if set true the message coming from backend can be directly shown to the client instead of parsing it
	Errors     []FieldError `json:"errors"`
	Action     *Action      `json:"action"`
}

func (e *HttpError) Error() string {
	return e.Message
}

func (e *HttpError) Is(target error) bool {
	_, ok := target.(*HttpError)
	return ok
}

func (e *HttpError) WithMessage(message string) *HttpError {
	return &HttpError{
		Code:       e.Code,
		Message:    message,
		Override:   e.Override,
		StatusCode: e.StatusCode,
		Errors:     e.Errors,
		Action:     e.Action,
	}
}

func MakeUpperCaseWithUnderscores(str string) string {
	return strings.ToUpper(strings.ReplaceAll(str, " ", "_"))
}
