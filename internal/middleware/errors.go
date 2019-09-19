package middleware

import (
	"github.com/noelruault/ratingsapp/internal/errors"
)

var (
	wrap  = errors.Wrapper("models")
	wrapi = errors.WrapInternal
)

// Errors that can be returned my middleware calls.
const (
	ErrForbidden     MiddlewareError = "middleware: forbidden, user does not have permissions to perform this action"
	ErrNotAcceptable MiddlewareError = "middleware: not_acceptable, the content-type provided is not supported or the requested accept header cannot be satisfied"
)

// MiddlewareError defines errors exported by this package. This type implement a Public() method that
// extracts a unique error code defined for each error value exported.
type MiddlewareError string

// Error returns the exact original message of the e value.
func (e MiddlewareError) Error() string {
	return string(e)
}

// Public extracts the error code string present on the value of e.
//
// An error code is defined as the string after the package prefix and colon, and before the comma that follows
// this string. Example:
//		"models: error_code, this is a validation error"
func (e MiddlewareError) Public() string {
	// remove the prefix
	s := string(e)[len("middleware: "):]

	// extract the error code
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			s = s[:i]
			break
		}
	}

	return s
}

type privateError string

func (e privateError) Error() string {
	return string(e)
}

// publicError defines an error that has public output, where the public output is
// an error code.
type publicError interface {
	Public() string
}
