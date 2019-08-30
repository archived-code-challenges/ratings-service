package middleware

import "github.com/gin-gonic/gin"

// This file contains general middleware used to set safe headers on all HTTP responses.

// SecureHeaders is a middleware that adds safe defaults to HTTP response headers.
//
// The following headers are set:
//
//		X-Content-Type-Options: nosniff
//		X-Frame-Options: deny
//		X-XSS-Protection: 1; mode=block
func SecureHeaders(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "deny")
	c.Writer.Header()["X-XSS-Protection"] = []string{"1; mode=block"}

	c.Next()
}
