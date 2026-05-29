package apperr

import (
	"errors"
	"fmt"
)

type ErrorType string

const (
	TypeValidation   ErrorType = "VALIDATION"
	TypeNotFound     ErrorType = "NOT_FOUND"
	TypeConflict     ErrorType = "CONFLICT"
	TypeUnauthorized ErrorType = "UNAUTHORIZED"
	TypeForbidden    ErrorType = "FORBIDDEN"
	TypeInternal     ErrorType = "INTERNAL"
)

type AppError struct {
	Type    ErrorType
	Code    string
	Message string
	Err     error
}

var _ error = (*AppError)(nil)

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s (%s): %v", e.Type, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s (%s)", e.Type, e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(errType ErrorType, code string, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewValidationError(code string, message string, err error) *AppError {
	return NewAppError(TypeValidation, code, message, err)
}

func NewNotFoundError(code string, message string) *AppError {
	return NewAppError(TypeNotFound, code, message, nil)
}

func NewConflictError(code string, message string, err error) *AppError {
	return NewAppError(TypeConflict, code, message, err)
}

func NewUnauthorizedError(code string, message string) *AppError {
	return NewAppError(TypeUnauthorized, code, message, nil)
}

func NewForbiddenError(code string, message string) *AppError {
	return NewAppError(TypeForbidden, code, message, nil)
}

func NewInternalError(code string, message string, err error) *AppError {
	return NewAppError(TypeInternal, code, message, err)
}

// Helper to determine if an error is of a specific AppError type
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

func GetErrorType(err error) ErrorType {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type
	}
	return TypeInternal
}

