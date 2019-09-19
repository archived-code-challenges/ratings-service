package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestModelError_Error(t *testing.T) {
	var cases = []struct {
		txt string
	}{
		{"models: test error number one"},
		{"models: another test error"},
		{"models: error_code, another test error"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.txt, ModelError(cs.txt).Error())
	}
}

func TestPrivateError_Error(t *testing.T) {
	var cases = []struct {
		txt string
	}{
		{"test error number one"},
		{"another test error"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.txt, privateError(cs.txt).Error())
	}

	type public interface{ Public() string }
	_, ok := interface{}(privateError("test error")).(public)
	assert.False(t, ok, "must not be a public error")

}

func TestModelError_Public(t *testing.T) {
	var cases = []struct {
		txt string
		out string
	}{
		{"models: number_one, test error number one", "number_one"},
		{"models: another_error, another test error", "another_error"},
		{"models: number_three, bad token received from you", "number_three"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.out, ModelError(cs.txt).Public())
	}
}

func TestValidationError_Error(t *testing.T) {
	var cases = []struct {
		txt ValidationError
		out []string
	}{
		{
			ValidationError{"field1": ModelError("skdjhfgklsjdfkhjg"), "field_2": ModelError("sdfg")},
			[]string{"models: validation error on fields", "field1", "field_2"},
		},
	}

	for _, cs := range cases {
		for _, s := range cs.out {
			assert.Contains(t, cs.txt.Error(), s)
		}
	}
}

func TestValidationError_Public(t *testing.T) {
	var cases = []struct {
		txt ValidationError
		out string
	}{
		{ValidationError{"field1": ModelError("skdjhfgklsjdfkhjg"), "field_2": ModelError("sdfg")}, "validation_error"},
		{ValidationError{}, "validation_error"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.out, cs.txt.Public())
	}
}

func TestValidationError_Is(t *testing.T) {
	var cases = []struct {
		name   string
		in     ValidationError
		target error
		out    bool
	}{
		{
			"notValidationError",
			ValidationError{},
			privateError("another type of error here"),
			false,
		},
		{
			"bothEmpty",
			ValidationError{},
			ValidationError{},
			true,
		},
		{
			"isSubset",
			ValidationError{
				"field1": ErrNoCredentials,
				"field2": ErrRequired,
				"field3": ErrTooShort,
			},
			ValidationError{
				"field1": ErrNoCredentials,
				"field3": ErrTooShort,
			},
			true,
		},
		{
			"intersects",
			ValidationError{
				"field1": ErrNoCredentials,
				"field2": ErrRequired,
				"field3": ErrTooShort,
			},
			ValidationError{
				"field1": ErrNoCredentials,
				"field3": ErrTooShort,
				"field5": ErrInUse,
			},
			false,
		},
		{
			"disjunct",
			ValidationError{
				"field1": ErrNoCredentials,
				"field2": ErrRequired,
				"field3": ErrTooShort,
			},
			ValidationError{
				"field4": ErrNoCredentials,
				"field5": ErrInUse,
			},
			false,
		},
		{
			"targetEmpty",
			ValidationError{
				"field1": ErrNoCredentials,
				"field2": ErrRequired,
				"field3": ErrTooShort,
			},
			ValidationError{},
			true,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			assert.Equal(t, cs.out, cs.in.Is(cs.target))
			assert.Equal(t, cs.out, xerrors.Is(cs.in, cs.target))
		})
	}
}
