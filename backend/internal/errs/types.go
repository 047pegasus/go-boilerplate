package errs

import "net/http"

func NewUnAuthorizedError(msg string, override bool) *HttpError {
	return &HttpError{
		Code:       MakeUpperCaseWithUnderscores(http.StatusText(http.StatusUnauthorized)),
		Message:    msg,
		StatusCode: http.StatusUnauthorized,
		Override:   override,
	}
}

func NewForbiddenError(msg string, override bool) *HttpError {
	return &HttpError{
		Code:       MakeUpperCaseWithUnderscores(http.StatusText(http.StatusForbidden)),
		Message:    msg,
		StatusCode: http.StatusForbidden,
		Override:   override,
	}
}

func NewBadRequestError(msg string, override bool, code *string, errors []FieldError, action *Action) *HttpError {
	formattedCode := MakeUpperCaseWithUnderscores(http.StatusText(http.StatusBadRequest))
	if code != nil {
		formattedCode = *code
	}
	return &HttpError{
		Code:       formattedCode,
		Message:    msg,
		StatusCode: http.StatusBadRequest,
		Override:   override,
		Action:     action,
		Errors:     errors,
	}
}

func NewNotFoundError(msg string, override bool, code *string) *HttpError {
	formattedCode := MakeUpperCaseWithUnderscores(http.StatusText(http.StatusNotFound))
	if code != nil {
		formattedCode = *code
	}
	return &HttpError{
		Code:       formattedCode,
		Message:    msg,
		StatusCode: http.StatusNotFound,
		Override:   override,
	}
}

func NewInternalServerError() *HttpError {
	return &HttpError{
		Code:       MakeUpperCaseWithUnderscores(http.StatusText(http.StatusInternalServerError)),
		Message:    http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
		Override:   false,
	}
}

func ValidationError(err error) *HttpError {
	return NewBadRequestError("ValidationFailed:"+err.Error(), false, nil, nil, nil)
}
