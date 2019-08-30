package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ContentType is a middleware that makes sure all request with input are provided with
// an appropriate Content-Type value, and if an Accept header is provided, it includes the
// type in ct.
func ContentType(ct string) gin.HandlerFunc {
	mime := strings.SplitN(ct, "/", 2)
	if len(mime) != 2 {
		panic(wrap("content type passed as input must be in the format xxxx/yyyyy", nil))
	}

	return func(c *gin.Context) {
		acc := c.GetHeader("Accept")
		if acc != "" &&
			!strings.Contains(acc, "*/*") &&
			!strings.Contains(acc, mime[0]+"/*") &&
			!strings.Contains(acc, ct) {
			viewErr.JSON(c, ErrNotAcceptable)
		}

		ctype := c.ContentType()
		if c.Request.Method == "POST" ||
			c.Request.Method == "PUT" ||
			c.Request.Method == "PATCH" {
			if !strings.Contains(ctype, ct) {
				viewErr.JSON(c, ErrNotAcceptable)
			}
		}

		c.Next()
	}
}
