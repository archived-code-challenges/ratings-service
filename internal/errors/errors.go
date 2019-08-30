/*
Package errors provides utilities that make it easier to wrap and unwrap errors
throughout the project.
*/
package errors

import "golang.org/x/xerrors"

// FuncWrap is a function that wraps the err argument with the msg message,
// returning the wrapped error. When err is nil, the function will create a new
// error.
type FuncWrap func(msg string, err error) error

// Wrapper generates an error wrapping function that prepends the package name
// to the error results.
func Wrapper(pkg string) FuncWrap {
	return func(msg string, err error) error {
		if err == nil {
			return xerrors.Errorf(pkg + ": " + msg)
		}

		return xerrors.Errorf(pkg+": "+msg+": %w", err)
	}
}

// WrapInternal wraps the err argument with the msg message without prepending
// any package information.
func WrapInternal(msg string, err error) error {
	if err == nil {
		return xerrors.Errorf(msg)
	}

	return xerrors.Errorf(msg+": %w", err)
}
