package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/stretchr/testify/assert"
)

type testUserService struct {
	models.UserService
	auth    func(username, password string) (models.User, error)
	refresh func(refreshToken string) (models.User, error)
	token   func(*models.User) (models.Token, error)
	byID    func(int64) (models.User, error)
	byIDs   func(...int64) ([]models.User, error)
	delete  func(int64) error
	create  func(*models.User) error
	update  func(*models.User) error
}

func (t *testUserService) Authenticate(username, password string) (models.User, error) {
	if t.auth != nil {
		return t.auth(username, password)
	}

	panic("not provided")
}

func (t *testUserService) Refresh(refreshToken string) (models.User, error) {
	if t.refresh != nil {
		return t.refresh(refreshToken)
	}

	panic("not provided")
}

func (t *testUserService) Token(u *models.User) (models.Token, error) {
	if t.token != nil {
		return t.token(u)
	}

	panic("not provided")
}

func (t *testUserService) ByID(id int64) (models.User, error) {
	if t.byID != nil {
		return t.byID(id)
	}

	panic("not provided")
}

func (t *testUserService) ByIDs(id ...int64) ([]models.User, error) {
	if t.byIDs != nil {
		return t.byIDs(id...)
	}

	panic("not provided")
}

func (t *testUserService) Delete(id int64) error {
	if t.delete != nil {
		return t.delete(id)
	}

	panic("not provided")
}

func (t *testUserService) Create(u *models.User) error {
	if t.create != nil {
		return t.create(u)
	}

	panic("not provided")
}

func (t *testUserService) Update(u *models.User) error {
	if t.update != nil {
		return t.update(u)
	}

	panic("not provided")
}

func TestUsers_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	us := &testUserService{}
	u := NewUsers(us)

	var cases = []struct {
		name        string
		contentType string
		content     string
		outStatus   int
		outJSON     string
		setup       func(*testing.T)
	}{
		{
			"badMIME",
			"asdfsdfg",
			"grant_type=password&username=user@example.com&password=1234luggage",
			http.StatusBadRequest,
			`{"error": "invalid_request", "error_description":"content_type_not_accepted"}`,
			nil,
		},
		{
			"badContent",
			"application/x-www-form-urlencoded",
			"graskdfhjglk!@98574sjdgfh ksdhf lksdfghlksjkl",
			http.StatusBadRequest,
			`{"error": "invalid_request", "error_description":"invalid_form"}`,
			nil,
		},
		{
			"badGrantTypeDevice",
			"application/x-www-form-urlencoded",
			"grant_type=device",
			http.StatusBadRequest,
			`{"error": "unsupported_grant_type"}`,
			nil,
		},
		{
			"badGrantTypeAuthCode",
			"application/x-www-form-urlencoded",
			"grant_type=authorization_code",
			http.StatusBadRequest,
			`{"error": "unsupported_grant_type"}`,
			nil,
		},
		{
			"validationFails",
			"application/x-www-form-urlencoded",
			"grant_type=password",
			http.StatusBadRequest,
			`{"error": "invalid_request", "error_description":"credentials_not_provided"}`,
			func(*testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					return models.User{}, models.ErrNoCredentials
				}
			},
		},
		{
			"authInternalError",
			"application/x-www-form-urlencoded",
			"grant_type=password",
			http.StatusInternalServerError,
			`{"error": "server_error"}`,
			func(*testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					return models.User{}, privateError("models: some type of internal error")
				}
			},
		},
		{
			"tokInternalError",
			"application/x-www-form-urlencoded",
			"grant_type=password",
			http.StatusInternalServerError,
			`{"error": "server_error"}`,
			func(*testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					return models.User{}, nil
				}
				us.token = func(*models.User) (models.Token, error) {
					return models.Token{}, privateError("models: some type of internal error")
				}
			},
		},
		{
			"unauthorisedBadPass",
			"application/x-www-form-urlencoded",
			"grant_type=password",
			http.StatusUnauthorized,
			`{"error": "invalid_client"}`,
			func(*testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"unauthorisedNoUser",
			"application/x-www-form-urlencoded",
			"grant_type=password",
			http.StatusUnauthorized,
			`{"error": "invalid_client"}`,
			func(*testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"unauthorisedBadEmail",
			"application/x-www-form-urlencoded",
			"grant_type=password",
			http.StatusUnauthorized,
			`{"error": "invalid_client"}`,
			func(*testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"grantedPassword",
			"application/x-www-form-urlencoded",
			"grant_type=password&email=user%40example.com&password=1234luggage",
			http.StatusOK,
			`{"access_token": "test access token", "refresh_token": "test token", "expires_in": 900, "token_type": "bearer"}`,
			func(t *testing.T) {
				us.auth = func(username, password string) (models.User, error) {
					assert.Equal(t, username, "user@example.com")
					assert.Equal(t, password, "1234luggage")

					return models.User{
						ID:       99,
						Email:    username,
						Password: password,
					}, nil
				}
				us.token = func(u *models.User) (models.Token, error) {
					assert.Equal(t, int64(99), u.ID)

					return models.Token{
						RefreshToken: "test token",
						AccessToken:  "test access token",
						ExpiresIn:    900,
						TokenType:    "bearer",
					}, nil
				}
			},
		},

		{
			"refreshValidationFails",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token",
			http.StatusBadRequest,
			`{"error": "invalid_request", "error_description":"credentials_not_provided"}`,
			func(*testing.T) {
				us.refresh = func(r string) (models.User, error) {
					return models.User{}, models.ErrNoCredentials
				}
			},
		},
		{
			"refreshInternalError",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token",
			http.StatusInternalServerError,
			`{"error": "server_error"}`,
			func(*testing.T) {
				us.refresh = func(r string) (models.User, error) {
					return models.User{}, privateError("models: some type of internal error")
				}
			},
		},
		{
			"tokRefreshInternalError",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token",
			http.StatusInternalServerError,
			`{"error": "server_error"}`,
			func(*testing.T) {
				us.refresh = func(r string) (models.User, error) {
					return models.User{}, privateError("models: some type of internal error")
				}
			},
		},
		{
			"unauthorisedExpiredToken",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token",
			http.StatusUnauthorized,
			`{"error": "invalid_client"}`,
			func(*testing.T) {
				us.refresh = func(r string) (models.User, error) {
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"refreshUnauthorisedNoUser",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token",
			http.StatusUnauthorized,
			`{"error": "invalid_client"}`,
			func(*testing.T) {
				us.refresh = func(r string) (models.User, error) {
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"unauthorisedBadToken",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token",
			http.StatusUnauthorized,
			`{"error": "invalid_client"}`,
			func(*testing.T) {
				us.refresh = func(r string) (models.User, error) {
					return models.User{}, models.ErrUnauthorised
				}
			},
		},
		{
			"grantedRefresh",
			"application/x-www-form-urlencoded",
			"grant_type=refresh_token&refresh_token=k%40sjdhdfgkjsgfkj",
			http.StatusOK,
			`{"access_token": "test access token", "refresh_token": "test token", "expires_in": 900, "token_type": "bearer"}`,
			func(t *testing.T) {
				us.refresh = func(r string) (models.User, error) {
					assert.Equal(t, r, "k@sjdhdfgkjsgfkj")

					return models.User{
						ID: 99,
					}, nil

				}
				us.token = func(u *models.User) (models.Token, error) {
					assert.Equal(t, int64(99), u.ID)

					return models.Token{
						RefreshToken: "test token",
						AccessToken:  "test access token",
						ExpiresIn:    900,
						TokenType:    "bearer",
					}, nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/v1/oauth/token",
				bytes.NewReader([]byte(cs.content)))
			c.Request.Header.Add("Content-Type", cs.contentType)

			if cs.setup != nil {
				cs.setup(t)
			}

			u.Login(c)

			res := w.Result()
			assert.Equal(t, cs.outStatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
			assert.JSONEq(t, cs.outJSON, w.Body.String())

			*us = testUserService{}
		})
	}
}

func TestUsers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	us := &testUserService{}
	u := NewUsers(us)

	mux := gin.New()
	mux.POST("/api/v1/users/", u.Create)

	var cases = []struct {
		name      string
		input     string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"notJSON",
			"a dalhd lkald fkjahd lfkjasdlf ",
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"validationError",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"email":"invalid","password":"required"}}`,
			func(t *testing.T) {
				us.create = func(u *models.User) error {
					user := models.NewUser()
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return models.ValidationError{
						"email":    models.ErrInvalid,
						"password": models.ErrRequired,
					}
				}
			},
		},
		{
			"internalError",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				us.create = func(u *models.User) error {
					user := models.NewUser()
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return privateError("test error message")
				}
			},
		},
		{
			"emailTaken",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusConflict,
			`{"error":"validation_error","fields":{"email":"is_duplicate","password":"required"}}`,
			func(t *testing.T) {
				us.create = func(u *models.User) error {
					user := models.NewUser()
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return models.ValidationError{
						"email":    models.ErrDuplicate,
						"password": models.ErrRequired,
					}
				}
			},
		},
		{
			"ok",
			`{"active":true,"email":"someone@somewhere.com",
				"firstName":"John","lastName":"Dear","password":"testpassword","roleId":99,
				"settings":"a string of preferences"}`,
			http.StatusCreated,
			`{"id":88,"active":true,"email":"someone@somewhere.com",
				"firstName":"John","lastName":"Dear","password":"testpassword","roleId":99,
				"settings":"a string of preferences"}`,
			func(t *testing.T) {
				us.create = func(u *models.User) error {
					assert.Equal(t, &models.User{
						Active:    true,
						Email:     "someone@somewhere.com",
						FirstName: "John",
						LastName:  "Dear",
						Password:  "testpassword",
						RoleID:    99,
						Settings:  "a string of preferences",
					}, u)

					u.ID = 88
					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/v1/users/",
				bytes.NewReader([]byte(cs.input)))
			c.Request.Header.Add("Accept", "application/json")
			c.Request.Header.Add("Content-Type", "application/json")

			if cs.setup != nil {
				cs.setup(t)
			}

			mux.HandleContext(c)

			res := w.Result()
			assert.Equal(t, cs.outStatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
			assert.JSONEq(t, cs.outJSON, w.Body.String())

			*us = testUserService{}
		})
	}

}

func TestUsers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)
	us := &testUserService{}
	u := NewUsers(us)

	mux := gin.New()
	mux.PUT("/api/v1/users/:id", u.Update)

	var cases = []struct {
		name      string
		path      string
		input     string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badPathID",
			"/api/v1/users/lksdjflk",
			"",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notJSON",
			"/api/v1/users/99",
			"a dalhd lkald fkjahd lfkjasdlf ",
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"notFound",
			"/api/v1/users/99",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				us.update = func(u *models.User) error {
					user := models.NewUser()
					user.ID = 99
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return models.ErrNotFound
				}
			},
		},
		{
			"validationError",
			"/api/v1/users/99",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"email":"invalid","password":"required"}}`,
			func(t *testing.T) {
				us.update = func(u *models.User) error {
					user := models.NewUser()
					user.ID = 99
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return models.ValidationError{
						"email":    models.ErrInvalid,
						"password": models.ErrRequired,
					}
				}
			},
		},
		{
			"cannotChangeAdmin",
			"/api/v1/users/1",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusConflict,
			`{"error":"read_only"}`,
			func(t *testing.T) {
				us.update = func(u *models.User) error {
					user := models.NewUser()
					user.ID = 1
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return models.ErrReadOnly
				}
			},
		},
		{
			"emailAlreadyUsed",
			"/api/v1/users/1",
			`{"email":"someone@somewhere.com","firstName":"John","lastName":"Dear"}`,
			http.StatusConflict,
			`{"error":"validation_error", "fields":{"email":"is_duplicate"}}`,
			func(t *testing.T) {
				us.update = func(u *models.User) error {
					user := models.NewUser()
					user.ID = 1
					user.Email = "someone@somewhere.com"
					user.FirstName = "John"
					user.LastName = "Dear"

					assert.Equal(t, &user, u)
					return models.ValidationError{"email": models.ErrDuplicate}
				}
			},
		},
		{
			"ok",
			"/api/v1/users/99",
			`{"active":true,"email":"someone@somewhere.com",
				"firstName":"John","lastName":"Dear","password":"testpassword","roleId":99,
				"settings":"a string of preferences"}`,
			http.StatusOK,
			`{"id":99,"active":true,"email":"someone@somewhere.com",
				"firstName":"John","lastName":"Dear","password":"testpassword","roleId":99,
				"settings":"a string of preferences"}`,
			func(t *testing.T) {
				us.update = func(u *models.User) error {
					assert.Equal(t, &models.User{
						ID:        99,
						Active:    true,
						Email:     "someone@somewhere.com",
						FirstName: "John",
						LastName:  "Dear",
						Password:  "testpassword",
						RoleID:    99,
						Settings:  "a string of preferences",
					}, u)

					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("PUT", cs.path,
				bytes.NewReader([]byte(cs.input)))
			c.Request.Header.Add("Accept", "application/json")
			c.Request.Header.Add("Content-Type", "application/json")

			if cs.setup != nil {
				cs.setup(t)
			}

			mux.HandleContext(c)

			res := w.Result()
			assert.Equal(t, cs.outStatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
			assert.JSONEq(t, cs.outJSON, w.Body.String())

			*us = testUserService{}
		})
	}
}

func TestUsers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	us := &testUserService{}
	u := NewUsers(us)

	mux := gin.New()
	mux.DELETE("/api/v1/users/:id", u.Delete)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badPathID",
			"/api/v1/users/lksdjflk",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/users/999",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				us.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return models.ErrNotFound
				}
			},
		},
		{
			"isAdmin",
			"/api/v1/users/999",
			http.StatusConflict,
			`{"error":"read_only"}`,
			func(t *testing.T) {
				us.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return models.ErrReadOnly
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/users/999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				us.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/users/999",
			http.StatusNoContent,
			`{}`,
			func(t *testing.T) {
				us.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("DELETE", cs.path, nil)
			c.Request.Header.Add("Accept", "application/json")

			if cs.setup != nil {
				cs.setup(t)
			}

			mux.HandleContext(c)

			res := w.Result()
			assert.Equal(t, cs.outStatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")

			if res.StatusCode != 204 {
				assert.JSONEq(t, cs.outJSON, w.Body.String())
			} else {
				assert.Equal(t, "", w.Body.String())
			}

			*us = testUserService{}
		})
	}
}

func TestUsers_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	us := &testUserService{}
	u := NewUsers(us)

	mux := gin.New()
	mux.GET("/api/v1/users/:id", u.Get)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badPathID",
			"/api/v1/users/lksdjflk",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/users/999",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				us.byID = func(id int64) (models.User, error) {
					assert.Equal(t, int64(999), id)
					return models.User{}, models.ErrNotFound
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/users/999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				us.byID = func(id int64) (models.User, error) {
					assert.Equal(t, int64(999), id)
					return models.User{}, wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/users/999",
			http.StatusOK,
			`{
				"active":true,
				"email":"test@email.com",
				"firstName":"Test",
				"id":999,
				"lastName":"User",
				"role":{
					"id":88,
					"label":"testrole",
					"permissions":[
						"readUsers",
						"writeUsers",
						"readRatings",
						"writeRatings"
					]
				},
				"roleId":88,
				"settings":"settings_string"
			}`,
			func(t *testing.T) {
				us.byID = func(id int64) (models.User, error) {
					assert.Equal(t, int64(999), id)
					return models.User{
						ID:        999,
						Active:    true,
						Email:     "test@email.com",
						FirstName: "Test",
						LastName:  "User",
						RoleID:    88,
						Role: &models.Role{
							ID:          88,
							Label:       "testrole",
							Permissions: 15,
						},
						Settings: "settings_string",
					}, nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", cs.path, nil)
			c.Request.Header.Add("Accept", "application/json")

			if cs.setup != nil {
				cs.setup(t)
			}

			mux.HandleContext(c)

			res := w.Result()
			assert.Equal(t, cs.outStatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
			assert.JSONEq(t, cs.outJSON, w.Body.String())

			*us = testUserService{}
		})
	}
}

func TestUsers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	us := &testUserService{}
	u := NewUsers(us)

	mux := gin.New()
	mux.GET("/api/v1/users/", u.List)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badQueryID",
			"/api/v1/users/?id=kdfjhgkhjlsfg",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"id": "invalid_parse"}}`,
			nil,
		},
		{
			"badQueryID2",
			"/api/v1/users/?id=kdfjhg,khjlsfg",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"id": "invalid_parse"}}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/users/?id=999,1000",
			http.StatusOK,
			`{"items":[]}`,
			func(t *testing.T) {
				us.byIDs = func(id ...int64) ([]models.User, error) {
					assert.Equal(t, []int64{999, 1000}, id)
					return nil, nil
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/users/?id=999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				us.byIDs = func(id ...int64) ([]models.User, error) {
					assert.Equal(t, []int64{999}, id)
					return nil, wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/users/?id=999,888",
			http.StatusOK,
			`{"items":[{
				"active":true,
				"email":"test@email.com",
				"firstName":"Test",
				"id":999,
				"lastName":"User",
				"role":{
					"id":88,
					"label":"testrole",
					"permissions":[
						"readUsers",
						"writeUsers",
						"readRatings",
						"writeRatings"
					]
				},
				"roleId":88,
				"settings":"settings_string"
			}]}`,
			func(t *testing.T) {
				us.byIDs = func(id ...int64) ([]models.User, error) {
					assert.Equal(t, []int64{999, 888}, id)
					return []models.User{
						{
							ID:        999,
							Active:    true,
							Email:     "test@email.com",
							FirstName: "Test",
							LastName:  "User",
							RoleID:    88,
							Role: &models.Role{
								ID:          88,
								Label:       "testrole",
								Permissions: 15,
							},
							Settings: "settings_string",
						},
					}, nil
				}
			},
		},
		{
			"okBlankID",
			"/api/v1/users/?id=",
			http.StatusOK,
			`{"items":[{
				"active":true,
				"email":"test@email.com",
				"firstName":"Test",
				"id":999,
				"lastName":"User",
				"role":{
					"id":88,
					"label":"testrole",
					"permissions":[
						"readUsers",
						"writeUsers",
						"readRatings",
						"writeRatings"
					]
				},
				"roleId":88,
				"settings":"settings_string"
			}]}`,
			func(t *testing.T) {
				us.byIDs = func(id ...int64) ([]models.User, error) {
					assert.Len(t, id, 0)
					return []models.User{
						{
							ID:        999,
							Active:    true,
							Email:     "test@email.com",
							FirstName: "Test",
							LastName:  "User",
							RoleID:    88,
							Role: &models.Role{
								ID:          88,
								Label:       "testrole",
								Permissions: 15,
							},
							Settings: "settings_string",
						},
					}, nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", cs.path, nil)
			c.Request.Header.Add("Accept", "application/json")

			if cs.setup != nil {
				cs.setup(t)
			}

			mux.HandleContext(c)

			res := w.Result()
			assert.Equal(t, cs.outStatus, res.StatusCode)
			assert.Contains(t, res.Header.Get("Content-Type"), "application/json")
			assert.JSONEq(t, cs.outJSON, w.Body.String())

			*us = testUserService{}
		})
	}
}
