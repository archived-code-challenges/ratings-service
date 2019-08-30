package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hdl := func(c *gin.Context) {
		c.JSON(200, gin.H{"test": "ok"})
	}

	var cases = []struct {
		name        string
		ctype       string
		atype       string
		expectctype string
		method      string
		outstatus   int
		outbody     string
	}{
		{
			"acceptNoMatch1",
			"",
			"text/plain",
			"application/json",
			"GET",
			http.StatusNotAcceptable,
			`{"error":"not_acceptable"}`,
		},
		{
			"acceptNoMatch2",
			"",
			"text/*",
			"application/json",
			"HEAD",
			http.StatusNotAcceptable,
			`{"error":"not_acceptable"}`,
		},
		{
			"acceptNoMatch2",
			"",
			"image/*",
			"application/json",
			"GET",
			http.StatusNotAcceptable,
			`{"error":"not_acceptable"}`,
		},
		{
			"acceptMatch1",
			"",
			"",
			"application/json",
			"GET",
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"acceptMatch2",
			"",
			"*/*",
			"application/json",
			"GET",
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"acceptMatch2",
			"",
			"application/*",
			"application/json",
			"GET",
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"acceptMatch2",
			"",
			"application/json",
			"application/json",
			"GET",
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"ctypeNoMatch",
			"text/html",
			"*/*",
			"application/json",
			"POST",
			http.StatusNotAcceptable,
			`{"error":"not_acceptable"}`,
		},
		{
			"ctypeMatch",
			"application/json; charset=utf8",
			"*/*",
			"application/json",
			"POST",
			http.StatusOK,
			`{"test":"ok"}`,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			mux := gin.New()
			mux.Use(ContentType(cs.expectctype))
			mux.GET("/", hdl)
			mux.POST("/", hdl)
			mux.PATCH("/", hdl)
			mux.DELETE("/", hdl)
			mux.PUT("/", hdl)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(cs.method, "/", nil)
			if cs.ctype != "" {
				c.Request.Header.Add("Content-Type", cs.ctype)
			}
			if cs.atype != "" {
				c.Request.Header.Add("Accept", cs.atype)
			}

			mux.HandleContext(c)

			assert.Equal(t, cs.outstatus, w.Code)
			assert.JSONEq(t, cs.outbody, w.Body.String())
		})
	}

	t.Run("contentTypeMIME", func(t *testing.T) {
		assert.Panics(t, func() {
			ContentType("justaword")
		}, "must not accept invalid mime types")
	})
}
