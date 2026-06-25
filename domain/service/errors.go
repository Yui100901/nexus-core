package service

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorKindBadRequest ErrorKind = "bad_request"
	ErrorKindNotFound   ErrorKind = "not_found"
	ErrorKindConflict   ErrorKind = "conflict"
	ErrorKindForbidden  ErrorKind = "forbidden"
	ErrorKindInternal   ErrorKind = "internal"
)

type AppError struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func ErrorKindOf(err error) ErrorKind {
	if err == nil {
		return ""
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Kind
	}
	return ErrorKindInternal
}

func ErrBadRequest(message string) error {
	return &AppError{Kind: ErrorKindBadRequest, Message: message}
}

func ErrNotFound(message string) error {
	return &AppError{Kind: ErrorKindNotFound, Message: message}
}

func ErrConflict(message string) error {
	return &AppError{Kind: ErrorKindConflict, Message: message}
}

func ErrForbidden(message string) error {
	return &AppError{Kind: ErrorKindForbidden, Message: message}
}

func ErrInternal(message string) error {
	return &AppError{Kind: ErrorKindInternal, Message: message}
}

func WrapInternal(message string, err error) error {
	if err == nil {
		return ErrInternal(message)
	}
	return &AppError{Kind: ErrorKindInternal, Message: message, Err: err}
}

func BadRequestf(format string, args ...interface{}) error {
	return ErrBadRequest(fmt.Sprintf(format, args...))
}

func Conflictf(format string, args ...interface{}) error {
	return ErrConflict(fmt.Sprintf(format, args...))
}

func Forbiddenf(format string, args ...interface{}) error {
	return ErrForbidden(fmt.Sprintf(format, args...))
}
