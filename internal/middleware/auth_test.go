package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/stretchr/testify/assert"
)

type testUserService struct {
	validate func(string) (models.User, error)
}

func (tus *testUserService) Validate(aToken string) (models.User, error) {
	if tus.validate != nil {
		return tus.validate(aToken)
	}

	return models.User{}, nil
}

func TestAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tus := testUserService{}
	hdl := func(c *gin.Context) {
		c.JSON(200, gin.H{"test": "ok"})
	}
	mux := gin.New()
	mux.Use(Authenticated(&tus))
	mux.GET("/", hdl)

	var cases = []struct {
		name      string
		header    string
		outstatus int
		outbody   string
		setup     func(t *testing.T)
	}{
		{
			"noHeader",
			"",
			http.StatusUnauthorized,
			`{"error":"unauthorised"}`,
			nil,
		},
		{
			"badHeader",
			"gibberishskjdgdfhj",
			http.StatusUnauthorized,
			`{"error":"unauthorised"}`,
			nil,
		},
		{
			"noToken",
			"Bearer ",
			http.StatusUnauthorized,
			`{"error":"unauthorised"}`,
			nil,
		},
		{
			"badToken",
			"Bearer gibberish",
			http.StatusUnauthorized,
			`{"error":"unauthorised"}`,
			func(t *testing.T) {
				tus.validate = func(atok string) (models.User, error) {
					assert.Equal(t, "gibberish", atok)
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"internalError",
			"Bearer sometoken",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				tus.validate = func(atok string) (models.User, error) {
					assert.Equal(t, "sometoken", atok)
					return models.User{}, errors.New("some internal type of error")
				}
			},
		},
		{
			"ok",
			"Bearer sometoken",
			http.StatusOK,
			`{"test":"ok"}`,
			func(t *testing.T) {
				tus.validate = func(atok string) (models.User, error) {
					assert.Equal(t, "sometoken", atok)
					return models.User{}, nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/", nil)
			if cs.header != "" {
				c.Request.Header.Add("Authorization", cs.header)
			}

			if cs.setup != nil {
				cs.setup(t)
			}

			mux.HandleContext(c)

			assert.Equal(t, cs.outstatus, w.Code)
			assert.JSONEq(t, cs.outbody, w.Body.String())

			user, ok := c.Get("user")
			if cs.outstatus != http.StatusOK {
				assert.False(t, ok, "does not have a new user set to the context")
			} else {
				assert.True(t, ok, "does have a new user set to the context")
				assert.IsType(t, &models.User{}, user)
			}
		})
	}
}

func TestCan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hdl := func(c *gin.Context) {
		c.JSON(200, gin.H{"test": "ok"})
	}

	var cases = []struct {
		name       string
		permission models.Permissions
		want       models.Permissions
		outstatus  int
		outbody    string
	}{
		{
			"canRead",
			0,
			0,
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"matchMustHaveAll",
			4,
			6,
			http.StatusForbidden,
			`{"error":"forbidden"}`,
		},
		{
			"matchExactly",
			15,
			15,
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"matchOne1",
			15,
			1,
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"matchOne2",
			15,
			8,
			http.StatusOK,
			`{"test":"ok"}`,
		},
		{
			"noMatch0",
			0,
			8,
			http.StatusForbidden,
			`{"error":"forbidden"}`,
		},
		{
			"noMatchNoZero",
			20,
			8,
			http.StatusForbidden,
			`{"error":"forbidden"}`,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/", nil)

			c.Set("user", &models.User{
				Role: &models.Role{
					ID:          99,
					Label:       "testrole",
					Permissions: cs.permission,
				},
			})
			Can(cs.want, hdl)(c)

			assert.Equal(t, cs.outstatus, w.Code)
			assert.JSONEq(t, cs.outbody, w.Body.String())
		})
	}
}
