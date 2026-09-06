package validation

import "C"
import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/047pegasus/go-boilerplate/internal/errs"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

type Validatable interface {
	Validate() error
}

type CustomValidationError struct {
	Field   string
	Message string
}
type CustomValidationErrors []CustomValidationError

func (c CustomValidationErrors) Error() string {
	return "validation failed"
}

func BindAndValidate(c *echo.Context, payload Validatable) error {
	if err := c.Bind(payload); err != nil {
		message := strings.Split(strings.Split(err.Error(), ",")[1], "message=")[1]
		return errs.NewBadRequestError(message, false, nil, nil, nil)
	}
	if msg, fieldErrors := validateStruct(payload); fieldErrors != nil {
		return errs.NewBadRequestError(msg, true, nil, fieldErrors, nil)
	}
	return nil
}

func BindAndValidateQuery(c *echo.Context, payload Validatable) error {
	if err := c.Bind(payload); err != nil {
		return errs.NewBadRequestError("Invalid query parameters", false, nil, nil, nil)
	}
	if msg, fieldErrors := validateStruct(payload); fieldErrors != nil {
		return errs.NewBadRequestError(msg, true, nil, fieldErrors, nil)
	}
	return nil
}

func validateStruct(payload Validatable) (string, []errs.FieldError) {
	if err := payload.Validate(); err != nil {
		return extractValidationErrors(err)
	}
	return "", nil
}

func extractValidationErrors(err error) (string, []errs.FieldError) {
	var fieldErrors []errs.FieldError
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		CustomValidationErrors := err.(CustomValidationErrors)
		for _, err := range CustomValidationErrors {
			fieldErrors = append(fieldErrors, errs.FieldError{
				Field: err.Field,
				Error: err.Message,
			})
		}
	}

	for _, err := range validationErrors {
		field := strings.ToLower(err.Field())
		var msg string
		switch err.Tag() {
		case "required":
			msg = "field is required"
		case "min":
			if err.Type().Kind() == reflect.String {
				msg = fmt.Sprintf("field must be at least %s characters", err.Param())
			} else {
				msg = fmt.Sprintf("must be at least %s ", err.Param())
			}
		case "max":
			if err.Type().Kind() == reflect.String {
				msg = fmt.Sprintf("field must not exceed %s characters", err.Param())
			} else {
				msg = fmt.Sprintf("must not exceed %s ", err.Param())
			}
		case "oneof":
			msg = fmt.Sprintf("field must be one of: %s", err.Param())
		case "email":
			msg = "must be a valid email address"
		case "e164":
			msg = "must be a valid phone number with a country code"
		case "uuid":
			msg = "must be a valid UUID"
		case "uuidList":
			msg = "must be a comma separated list of valid UUIDs"
		case "dive":
			msg = "some items are invalid"
		default:
			if err.Param() != "" {
				msg = fmt.Sprintf("%s: %s:%s", field, err.Tag(), err.Param())
			} else {
				msg = fmt.Sprintf("%s: %s", field, err.Tag())
			}
		}
		fieldErrors = append(fieldErrors, errs.FieldError{
			Field: strings.ToLower(err.Field()),
			Error: msg,
		})
	}
	return "Validation failed", fieldErrors
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isValidUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}
