package views

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/noelruault/ratingsapp/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var ev Error
	var cases = []struct {
		name      string
		seterror  models.PublicError
		setcode   int
		inerror   error
		outstatus int
		outjson   string
	}{
		{
			"appliesDefaultBadRequest",
			nil,
			0,
			models.ErrNoCredentials,
			http.StatusBadRequest,
			`{"error":"credentials_not_provided"}`,
		},
		{
			"appliesDefaultInternalError",
			nil,
			0,
			xerrors.Errorf("test error private"),
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
		},
		{
			"appliesCustomCode1",
			models.ErrNotFound,
			404,
			models.ErrNotFound,
			http.StatusNotFound,
			`{"error":"not_found"}`,
		},
		{
			"appliesCustomCode2",
			models.ErrReadOnly,
			409,
			models.ErrReadOnly,
			http.StatusConflict,
			`{"error":"read_only"}`,
		},
		{
			"validationErrors",
			nil,
			0,
			models.ValidationError{"field1": models.ErrInvalid, "field2": models.ErrTooShort},
			http.StatusBadRequest,
			`{"error":"validation_error", "fields":{"field1":"invalid", "field2":"too_short"}}`,
		},
		{
			"validationErrorsWithSpecialCode",
			models.ErrInvalid,
			409,
			models.ValidationError{"field1": models.ErrInvalid, "field2": models.ErrTooShort},
			http.StatusConflict,
			`{"error":"validation_error", "fields":{"field1":"invalid", "field2":"too_short"}}`,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			if cs.seterror != nil {
				ev.SetCode(cs.seterror, cs.setcode)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/v1/dummy/", nil)
			c.Request.Header.Add("Content-Type", "application/json")

			ev.JSON(c, cs.inerror)

			res := w.Result()
			assert.Equal(t, cs.outstatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
			assert.JSONEq(t, cs.outjson, w.Body.String())
		})
	}
}
