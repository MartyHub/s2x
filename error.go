package s2x

import "fmt"

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(format string, args ...any) *Error {
	return &Error{
		Message: fmt.Sprintf(format, args...),
	}
}
