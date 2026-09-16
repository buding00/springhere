package clierr

import (
	"errors"
	"fmt"
)

const (
	ExitOK        = 0
	ExitUsage     = 2
	ExitComponent = 3
	ExitCombine   = 4
	ExitContract  = 5
	ExitSource    = 6
	ExitConflict  = 7
	ExitGit       = 8
	ExitInternal  = 1
)

type Error struct {
	Code    int
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func Usage(msg string) error { return &Error{Code: ExitUsage, Message: msg} }
func Usagef(f string, a ...any) error {
	return Usage(fmt.Sprintf(f, a...))
}
func Component(msg string) error { return &Error{Code: ExitComponent, Message: msg} }
func Combine(msg string) error   { return &Error{Code: ExitCombine, Message: msg} }
func Contract(msg string, cause error) error {
	return &Error{Code: ExitContract, Message: msg, Cause: cause}
}
func Source(msg string, cause error) error {
	return &Error{Code: ExitSource, Message: msg, Cause: cause}
}
func Conflict(msg string) error { return &Error{Code: ExitConflict, Message: msg} }
func Git(msg string, cause error) error {
	return &Error{Code: ExitGit, Message: msg, Cause: cause}
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ce *Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ExitInternal
}
