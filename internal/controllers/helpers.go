package controllers

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
)

// parseForm uses gin to parse a form-encoded input into a destination value.
func parseForm(c *gin.Context, dst interface{}) error {
	if !strings.Contains(c.ContentType(), "application/x-www-form-urlencoded") {
		return ErrContentTypeNotAccepted
	}

	err := c.ShouldBind(dst)
	if err != nil {
		return ErrInvalidFormInput
	}

	return nil
}

// parseJSON uses Gin to parse a json input into a destination value. It does not check for
// Content-Type as it is expected to already be checked by middleware.
func parseJSON(c *gin.Context, dst interface{}) error {
	err := c.ShouldBindJSON(dst)
	if err != nil {
		return ErrInvalidJSONInput
	}

	return nil
}

// getParamInt retrieves an int64 parameter from the URL path of a request. In
// case the parameter is not an integer, ErrNotFound is returned.
// Negative integers are accepted.
func getParamInt(c *gin.Context, paramName string) (int64, error) {
	p := c.Param(paramName)

	v, err := strconv.ParseInt(p, 10, 0)
	if err != nil {
		return 0, ErrNotFound
	}

	return v, nil
}

// getQueryListInt retrieves a list of int64 parameters from a request's query string.
// In case paramName is not found, nil is returned for both return values.
// If there's a failure in parsing the integers in the list, a ValidationError
// is returned.
func getQueryListInt(c *gin.Context, paramName string) ([]int64, error) {
	p := c.Query(paramName)
	if p == "" {
		return nil, nil
	}

	sids := strings.Split(p, ",")

	var ret = make([]int64, 0, len(sids))
	for i := range sids {
		v, err := strconv.ParseInt(sids[i], 10, 0)
		if err != nil {
			return nil, models.ValidationError{
				"id": ErrParseError,
			}
		}

		ret = append(ret, v)
	}

	return ret, nil
}

// getEncodedListInt retrieves a list of int64 parameters from a query string.
// In case paramName is not found, nil is returned for both return values.
// If there's a failure in parsing the integers in the list, a ValidationError
// is returned.
func getEncodedListInt(queryString, paramName string) ([]int64, error) {
	values, err := url.ParseQuery(queryString)
	if err != nil {
		return nil, ErrParseError
	}

	p := values.Get(paramName)
	if p == "" {
		return nil, nil
	}

	sids := strings.Split(p, ",")

	var ret = make([]int64, 0, len(sids))
	for i := range sids {
		v, err := strconv.ParseInt(sids[i], 10, 0)
		if err != nil {
			return nil, models.ValidationError{
				"id": ErrParseError,
			}
		}

		ret = append(ret, v)
	}

	return ret, nil
}

// getQueryParam retrieves an int64 parameter from the URL path of a request.
// In case the parameter is not an integer, ErrNotFound is returned.
// Negative integers are accepted.
func getQueryParam(c *gin.Context, paramName string) (int64, error) {
	p := c.Query("target")

	id, err := strconv.ParseInt(p, 10, 0)
	if err != nil {
		return 0, models.ValidationError{
			"target": ErrParseError,
		}
	}

	return id, nil
}
