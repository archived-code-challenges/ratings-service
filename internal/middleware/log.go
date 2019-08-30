package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Log improves on Gin's logging middleware by including any error messages that may
// result from a request.
func Log(c *gin.Context) {
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery
	if raw != "" {
		path = path + "?" + raw
	}

	// Process request
	start := time.Now()
	c.Next()
	latency := time.Now().Sub(start)

	// log it
	logrus.WithFields(logrus.Fields{
		"status":  c.Writer.Status(),
		"latency": latency,
		"from":    c.ClientIP(),
		"method":  c.Request.Method,
		"path":    path,
		"comment": c.Errors.Errors(),
	}).Info("Gin Request")
}
