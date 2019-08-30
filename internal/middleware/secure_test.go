package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecureHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/snow/oauth/callback?code=testcode&state=teststate", nil)

	SecureHeaders(c)

	assert.Equal(t, "nosniff", c.Writer.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "deny", c.Writer.Header().Get("X-Frame-Options"))
	assert.Equal(t, []string{"1; mode=block"}, c.Writer.Header()["X-XSS-Protection"])
}
