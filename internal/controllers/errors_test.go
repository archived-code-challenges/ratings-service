package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestControllerError_Error(t *testing.T) {
	var cases = []struct {
		txt string
	}{
		{"controllers: test error number one"},
		{"controllers: another test error"},
		{"controllers: error_code, another test error"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.txt, ControllerError(cs.txt).Error())
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

func TestControllerError_Public(t *testing.T) {
	var cases = []struct {
		txt string
		out string
	}{
		{"controllers: number_one, test error number one", "number_one"},
		{"controllers: another_error, another test error", "another_error"},
		{"controllers: number_three, bad token received from you", "number_three"},
	}

	for _, cs := range cases {
		assert.Equal(t, cs.out, ControllerError(cs.txt).Public())
	}
}
