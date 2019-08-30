package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMiddlewareError_Error(t *testing.T) {
	var cases = []struct {
		txt string
	}{
		{"middleware: test error number one"},
		{"middleware: another test error"},
		{"middleware: error_code, another test error"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.txt, MiddlewareError(cs.txt).Error())
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

func TestMiddlewareError_Public(t *testing.T) {
	var cases = []struct {
		txt string
		out string
	}{
		{"middleware: number_one, test error number one", "number_one"},
		{"middleware: another_error, another test error", "another_error"},
		{"middleware: number_three, bad token received from you", "number_three"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.out, MiddlewareError(cs.txt).Public())
	}
}
