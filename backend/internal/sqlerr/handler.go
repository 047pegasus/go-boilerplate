package sqlerr

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/047pegasus/go-boilerplate/internal/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ErrCode reports the error code for a given error.
// If the error is nil or is not of type *Error it reports sqlerr.Other.
func ErrCode(err error) Code {
	var pgerr *Error
	if errors.As(err, &pgerr) {
		return pgerr.Code
	}
	return Other
}

// converts pg err to  our custom Error type defined in the sqlerr/error
func ConvertPgError(src *pgconn.PgError) *Error {
	return &Error{
		Code:           MapCode(src.Code),
		Severity:       MapSeverity(src.Severity),
		DatabaseCode:   src.Code,
		Message:        src.Message,
		SchemaName:     src.SchemaName,
		TableName:      src.TableName,
		ColumnName:     src.ColumnName,
		DataTypeName:   src.DataTypeName,
		ConstraintName: src.ConstraintName,
		driverErr:      src,
	}
}

// generateErrorCode creates consistent error codes from database errors
func GenerateErrorCode(tableName string, errType Code) string {
	if tableName == "" {
		tableName = "RECORD"
	}

	domain := strings.ToUpper(tableName)
	//singularize the table name
	if strings.HasSuffix(domain, "S") && len(domain) > 1 {
		domain = domain[:len(domain)-1]
	}
	action := "ERROR"
	switch errType {
	case ForeignKeyViolation:
		action = "NOT_FOUND"
	case UniqueViolation:
		action = "ALREADY_EXISTS"
	case NotNullViolation:
		action = "REQUIRED"
	case CheckViolation:
		action = "INVALID"
	}

	return fmt.Sprintf("%s_%s", domain, action)
}

func humanizeText(text string) string {
	if text == "" {
		return ""
	}
	return cases.Title(language.English).String(strings.ReplaceAll(text, "_", " "))
}

// this extracts entity name from db info
func GetEntityName(tableName string, colName string) string {
	//first priority is column name itself (most reliable for FK relations
	if colName != "" && strings.HasSuffix(strings.ToLower(colName), "_id") {
		entity := strings.TrimSuffix(strings.ToLower(colName), "_id")
		return humanizeText(entity)
	}
	//second priority is table name: fallback option
	if tableName != "" {
		entity := tableName
		//again singularize
		if strings.HasSuffix(entity, "s") && len(entity) > 1 {
			entity = entity[:len(entity)-1]
		}
		return humanizeText(entity)
	}

	//default hardcode fallback
	return "record"
}

// this generates a user-friendly error message
func formatUserFriendlyErrMsg(sqlErr *Error) string {
	entityName := GetEntityName(sqlErr.TableName, sqlErr.ColumnName)
	switch sqlErr.Code {
	case ForeignKeyViolation:
		return fmt.Sprintf("The referenced %s does not exist", entityName)
	case UniqueViolation:
		return fmt.Sprintf("A %s with this identifier already exists", entityName)
	case NotNullViolation:
		fieldName := humanizeText(sqlErr.ColumnName)
		if fieldName == "" {
			fieldName = "field"
		}
		return fmt.Sprintf("The %s is required", fieldName)
	case CheckViolation:
		fieldName := humanizeText(sqlErr.ColumnName)
		if fieldName != "" {
			return fmt.Sprintf("The %s value does not meet required conditions", fieldName)
		}
		return "One or more values do not meet required conditions"
	default:
		return "An error occurred while processing your request"
	}
}

// this gets field name from unique constraint
func extractColForUniqueViolation(constraintName string) string {
	if constraintName != "" {
		return ""
	}
	//standard naming convention
	if strings.HasPrefix(constraintName, "unique_") {
		parts := strings.Split(constraintName, "_")
		if len(parts) >= 3 {
			return parts[len(parts)-1]
		}
	}

	//try alt convention(table_col_key)
	re := regexp.MustCompile(`_([^_]+)_(?:key|ukey)$`)
	matches := re.FindStringSubmatch(constraintName)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// this processes a DB errr into an apt application err
func HandleError(err error) error {
	//return as is if its alr a custom http err
	var httpErr *errs.HttpError
	if errors.As(err, &httpErr) {
		return err
	}

	//handle pgx specific errors
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		sqlErr := ConvertPgError(pgerr)

		// generate an apt err code & msg
		errCode := GenerateErrorCode(sqlErr.TableName, sqlErr.Code)
		usrMsg := formatUserFriendlyErrMsg(sqlErr)

		switch sqlErr.Code {
		case ForeignKeyViolation:
			return errs.NewBadRequestError(usrMsg, false, &errCode, nil, nil)

		case UniqueViolation:
			columnName := extractColForUniqueViolation(sqlErr.ConstraintName)
			if columnName != "" {
				usrMsg = strings.ReplaceAll(usrMsg, "identifier", humanizeText(columnName))
			}
			return errs.NewBadRequestError(usrMsg, true, &errCode, nil, nil)

		case NotNullViolation:
			fieldErrors := []errs.FieldError{
				{
					Field: strings.ToLower(sqlErr.ColumnName),
					Error: "is required",
				},
			}
			return errs.NewBadRequestError(usrMsg, true, &errCode, fieldErrors, nil)

		case CheckViolation:
			return errs.NewBadRequestError(usrMsg, true, &errCode, nil, nil)

		default:
			return errs.NewInternalServerError()
		}
	}

	//handle common pgx errors
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, sql.ErrNoRows):
		errMsg := err.Error()
		tablePrefix := "table:"
		if strings.Contains(errMsg, tablePrefix) {
			table := strings.Split(strings.Split(errMsg, tablePrefix)[1], ":")[0]
			entityName := GetEntityName(table, "")
			return errs.NewNotFoundError(fmt.Sprintf("%s not found", entityName), true, nil)
		}
		return errs.NewNotFoundError("Resource not found", false, nil)
	}

	return errs.NewInternalServerError()
}
