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

type testRoleService struct {
	models.RoleService
	create func(*models.Role) error
	update func(*models.Role) error
	delete func(int64) error
	byID   func(int64) (models.Role, error)
	byIDs  func(...int64) ([]models.Role, error)
}

func (t *testRoleService) Create(mr *models.Role) error {
	if t.create != nil {
		return t.create(mr)
	}

	panic("not provided")
}

func (t *testRoleService) Update(mr *models.Role) error {
	if t.update != nil {
		return t.update(mr)
	}

	panic("not provided")
}

func (t *testRoleService) Delete(id int64) error {
	if t.delete != nil {
		return t.delete(id)
	}

	panic("not provided")
}

func (t *testRoleService) ByID(id int64) (models.Role, error) {
	if t.byID != nil {
		return t.byID(id)
	}

	panic("not provided")
}

func (t *testRoleService) ByIDs(id ...int64) ([]models.Role, error) {
	if t.byIDs != nil {
		return t.byIDs(id...)
	}

	panic("not provided")
}

func TestRoles_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRoleService{}
	r := NewRoles(rs)

	mux := gin.New()
	mux.POST("/api/v1/roles/", r.Create)

	var cases = []struct {
		name      string
		content   string
		outStatus int
		outJSON   string
		setup     func(t *testing.T)
	}{
		{
			"badContent",
			"graskdfhjglk!@98574sjdgfh ksdhf lksdfghlksjkl",
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"badContent2",
			`{
				label: asldfkj,
				permissions: asldfkj
			}`,
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"internalError",
			`{
				"label": "accountant",
				"permissions":	[
					"readUsers",
					"readRatings"
				]
			}`,
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.create = func(mr *models.Role) error {
					return privateError("test error message")
				}
			},
		},
		{
			"validationError",
			`{
				"label": "tes",
				"permissions":	[
					"readRatings"
				]
			}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"label":"too_short"}}`,
			func(t *testing.T) {
				rs.create = func(r *models.Role) error {
					role := models.NewRole()
					role.Label = "tes"
					role.Permissions = models.PermissionReadRatings

					assert.Equal(t, &role, r)
					return models.ValidationError{
						"label": models.ErrTooShort,
					}
				}
			},
		},
		{
			"labelTaken",
			`{
				"label": "testlabel",
				"permissions":	[
					"readRatings"
				]
			}`,
			http.StatusConflict,
			`{"error":"validation_error","fields":{"label":"is_duplicate"}}`,
			func(t *testing.T) {
				rs.create = func(r *models.Role) error {
					role := models.NewRole()
					role.Label = "testlabel"
					role.Permissions = models.PermissionReadRatings

					assert.Equal(t, &role, r)
					return models.ValidationError{
						"label": models.ErrDuplicate,
					}
				}
			},
		},
		{
			"ok",
			`{
				"label": "testlabel",
				"permissions":	[
					"readUsers",
					"readRatings"
				]
			}`,
			http.StatusCreated,
			`{
				"id": 99,
				"label": "testlabel",
				"permissions":	[
					"readUsers",
					"readRatings"
				]
			}`,
			func(t *testing.T) {
				rs.create = func(mr *models.Role) error {
					mr.ID = 99
					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/v1/roles/",
				bytes.NewReader([]byte(cs.content)))
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

			*rs = testRoleService{}
		})
	}
}

func TestRoles_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRoleService{}
	u := NewRoles(rs)

	mux := gin.New()
	mux.PUT("/api/v1/roles/:id", u.Update)

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
			"/api/v1/roles/lksdjflk",
			"",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notJSON",
			"/api/v1/roles/99",
			"a dalhd lkald fkjahd lfkjasdlf ",
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"notFound",
			"/api/v1/roles/99",
			`{"label":"atest","permissions":[]}`,
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				rs.update = func(r *models.Role) error {
					role := models.NewRole()
					role.ID = 99
					role.Label = "atest"

					assert.Equal(t, &role, r)
					return models.ErrNotFound
				}
			},
		},
		{
			"validationError",
			"/api/v1/roles/99",
			`{"label":"atest","permissions":["readRatings"]}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"label":"too_short"}}`,
			func(t *testing.T) {
				rs.update = func(r *models.Role) error {
					role := models.NewRole()
					role.ID = 99
					role.Label = "atest"
					role.Permissions = models.PermissionReadRatings

					assert.Equal(t, &role, r)
					return models.ValidationError{
						"label": models.ErrTooShort,
					}
				}
			},
		},
		{
			"labelTaken",
			"/api/v1/roles/99",
			`{"label":"admin","permissions":["readRatings"]}`,
			http.StatusConflict,
			`{"error":"validation_error","fields":{"label":"is_duplicate"}}`,
			func(t *testing.T) {
				rs.update = func(r *models.Role) error {
					role := models.NewRole()
					role.ID = 99
					role.Label = "admin"
					role.Permissions = models.PermissionReadRatings

					assert.Equal(t, &role, r)
					return models.ValidationError{
						"label": models.ErrDuplicate,
					}
				}
			},
		},
		{
			"ok",
			"/api/v1/roles/99",
			`{"label":"atest","permissions":["readRatings"]}`,
			http.StatusOK,
			`{"id": 99, "label":"atest","permissions":["readRatings"]}`,
			func(t *testing.T) {
				rs.update = func(r *models.Role) error {
					assert.Equal(t, &models.Role{
						ID:          99,
						Label:       "atest",
						Permissions: models.PermissionReadRatings,
					}, r)

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

			*rs = testRoleService{}
		})
	}
}

func TestRoles_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRoleService{}
	r := NewRoles(rs)

	mux := gin.New()
	mux.DELETE("/api/v1/roles/:id", r.Delete)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(t *testing.T)
	}{
		{
			"badPathID",
			"/api/v1/roles/lksdjflk",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/roles/999",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				rs.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return models.ErrNotFound
				}
			},
		},
		{
			"isAdminIsUser",
			"/api/v1/roles/999",
			http.StatusConflict,
			`{"error":"read_only"}`,
			func(t *testing.T) {
				rs.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return models.ErrReadOnly
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/roles/999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.delete = func(id int64) error {
					assert.Equal(t, int64(999), id)
					return wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/roles/999",
			http.StatusNoContent,
			`{}`,
			func(t *testing.T) {
				rs.delete = func(id int64) error {
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

			*rs = testRoleService{}
		})
	}
}

func TestRoles_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRoleService{}
	r := NewRoles(rs)

	mux := gin.New()
	mux.GET("/api/v1/roles/:id", r.Get)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badPathID",
			"/api/v1/roles/lksdjflk",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/roles/999",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				rs.byID = func(id int64) (models.Role, error) {
					assert.Equal(t, int64(999), id)
					return models.Role{}, models.ErrNotFound
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/roles/999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.byID = func(id int64) (models.Role, error) {
					assert.Equal(t, int64(999), id)
					return models.Role{}, wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/roles/2",
			http.StatusOK,
			`{
				"id":2,
				"label":"user",
				"permissions":["readUsers", "writeUsers", "readRatings"]
			}`,
			func(t *testing.T) {
				rs.byID = func(id int64) (models.Role, error) {
					return models.Role{
						ID:          2,
						Label:       "user",
						Permissions: models.PermissionReadUsers | models.PermissionWriteUsers | models.PermissionReadRatings,
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

			*rs = testRoleService{}
		})
	}
}

func TestRoles_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRoleService{}
	r := NewRoles(rs)

	mux := gin.New()
	mux.GET("/api/v1/roles/", r.List)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badQueryID",
			"/api/v1/roles/?id=kdfjhgkhjlsfg",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"id":"invalid_parse"}}`,
			nil,
		},
		{
			"badQueryID2",
			"/api/v1/roles/?id=kdfjhg,khjlsfg",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"id": "invalid_parse"}}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/roles/?id=999,1000",
			http.StatusOK,
			`{"items":[]}`,
			func(t *testing.T) {
				rs.byIDs = func(id ...int64) ([]models.Role, error) {
					assert.Equal(t, []int64{999, 1000}, id)
					return nil, nil
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/roles/?id=999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.byIDs = func(id ...int64) ([]models.Role, error) {
					assert.Equal(t, []int64{999}, id)
					return nil, wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/roles/?id=1,2",
			http.StatusOK,
			`{
				"items":[
				{
					"id":1,
					"label":"test1role",
					"permissions":["writeRatings"]
				},
				{
					"id":2,
					"label":"test2role",
					"permissions":["readUsers"]
				}]
			}`,
			func(t *testing.T) {
				rs.byIDs = func(id ...int64) ([]models.Role, error) {
					assert.Equal(t, []int64{1, 2}, id)
					return []models.Role{
							models.Role{
								ID:          1,
								Label:       "test1role",
								Permissions: models.PermissionWriteRatings,
							},
							models.Role{
								ID:          2,
								Label:       "test2role",
								Permissions: models.PermissionReadUsers,
							},
						},
						nil
				}
			},
		},
		{
			"okBlankID",
			"/api/v1/roles/?id=",
			http.StatusOK,
			`{
				"items":[
				{
					"id":1,
					"label":"test1role",
					"permissions":["writeRatings"]
				},
				{
					"id":2,
					"label":"test2role",
					"permissions":["readUsers"]
				}]
			}`,
			func(t *testing.T) {
				rs.byIDs = func(id ...int64) ([]models.Role, error) {
					assert.Len(t, id, 0)
					return []models.Role{
							models.Role{
								ID:          1,
								Label:       "test1role",
								Permissions: models.PermissionWriteRatings,
							},
							models.Role{
								ID:          2,
								Label:       "test2role",
								Permissions: models.PermissionReadUsers,
							},
						},
						nil
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

			*rs = testRoleService{}
		})
	}
}
