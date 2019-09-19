package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStatic_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sc := NewStatic()

	t.Run("json", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/snow/oauth/callback?code=testcode&state=teststate", nil)
		c.Request.Header.Add("Accept", "application/json")

		sc.NotFound(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.JSONEq(t, `{"error":"not_found"}`, w.Body.String())
	})
}

func TestStatic_Home(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sc := NewStatic()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	sc.Home(c)

	// TODO: this needs to be implemented
	t.Skip()

}
