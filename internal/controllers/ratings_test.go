package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/stretchr/testify/assert"
)

type testRatingService struct {
	models.RatingService
	create   func(*models.Rating) error
	update   func(*models.Rating) error
	delete   func(*models.Rating) error
	byID     func(int64) (models.Rating, error)
	byTarget func(int64) ([]models.Rating, error)
}

func (t *testRatingService) Create(mr *models.Rating) error {
	if t.create != nil {
		return t.create(mr)
	}

	panic("not provided")
}

func (t *testRatingService) Update(mr *models.Rating) error {
	if t.update != nil {
		return t.update(mr)
	}

	panic("not provided")
}

func (t *testRatingService) Delete(mr *models.Rating) error {
	if t.delete != nil {
		return t.delete(mr)
	}

	panic("not provided")
}

func (t *testRatingService) ByID(id int64) (models.Rating, error) {
	if t.byID != nil {
		return t.byID(id)
	}

	panic("not provided")
}

func (t *testRatingService) ByTarget(id int64) ([]models.Rating, error) {
	if t.byTarget != nil {
		return t.byTarget(id)
	}

	panic("not provided")
}

func TestRatings_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRatingService{}
	r := NewRatings(rs)

	mux := gin.New()
	mux.POST("/api/v1/ratings/", func(c *gin.Context) {
		c.Set("user", &models.User{
			ID: 1,
		})
	}, r.Create)

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
				score: asldfkj,
				target: asldfkj
			}`,
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"internalError",
			`{
				"score": 10,
				"target": 9999
			}`,
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.create = func(mr *models.Rating) error {
					return privateError("test error message")
				}
			},
		},
		{
			"validationError",
			`{
				"score": 10,
				"target": -9999
			}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target":"invalid"}}`,
			func(t *testing.T) {
				rs.create = func(mr *models.Rating) error {
					mr.UserID = 1
					mr.User = nil

					rating := models.NewRating()
					rating.Score = 10
					rating.Target = -9999
					rating.UserID = 1

					assert.Equal(t, &rating, mr)
					return models.ValidationError{
						"target": models.ErrInvalid,
					}
				}
			},
		},
		{
			"ok",
			`{
				"score": 10,
				"target": 9999
			}`,
			http.StatusCreated,
			`{
				"id": 99,
				"active": true,
				"anonymous": true,
				"extra": {},
				"date": 0,
				"score": 10,
				"target": 9999,
				"userId": 1
			}`,
			func(t *testing.T) {
				rs.create = func(mr *models.Rating) error {
					mr.ID = 99
					mr.UserID = 1
					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/v1/ratings/",
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

			*rs = testRatingService{}
		})
	}
}

func TestRatings_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRatingService{}
	r := NewRatings(rs)

	mux := gin.New()
	mux.PUT("/api/v1/ratings/:id", func(c *gin.Context) {
		c.Set("user", &models.User{
			ID: 1,
		})
	}, r.Update)

	var cases = []struct {
		name      string
		path      string
		content   string
		outStatus int
		outJSON   string
		setup     func(t *testing.T)
	}{
		{
			"badPathID",
			"/api/v1/ratings/lksdjflk",
			"",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"badContent",
			"api/v1/ratings/99",
			"graskdfhjglk!@98574sjdgfh ksdhf lksdfghlksjkl",
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"badContent2",
			"api/v1/ratings/99",
			`{
					score: asldfkj,
					target: asldfkj
				}`,
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"internalError",
			"api/v1/ratings/99",
			`{
					"score": 10,
					"target": 9999
				}`,
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.update = func(mr *models.Rating) error {
					return privateError("test error message")
				}
			},
		},
		{
			"validationError",
			"api/v1/ratings/99",
			`{
					"score": 10,
					"target": -9999
				}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target":"invalid"}}`,
			func(t *testing.T) {
				rs.update = func(mr *models.Rating) error {
					mr.UserID = 1
					mr.User = nil

					rating := models.NewRating()
					rating.ID = 99
					rating.Score = 10
					rating.Target = -9999
					rating.UserID = 1

					assert.Equal(t, &rating, mr)
					return models.ValidationError{
						"target": models.ErrInvalid,
					}
				}
			},
		},
		{
			"notFound",
			"/api/v1/ratings/99",
			`{"score": 10, "target": 9999}`,
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				rs.update = func(mr *models.Rating) error {

					mr.UserID = 1
					mr.User = nil

					rating := models.NewRating()
					rating.ID = 99
					rating.Score = 10
					rating.Target = 9999
					rating.UserID = 1

					assert.Equal(t, &rating, mr)
					return models.ErrNotFound
				}
			},
		},
		{
			"ok",
			"api/v1/ratings/99",
			`{
				"comment": "a great comment",
				"extra": {"color": "red"},
				"score": 10,
				"target": 9999
			}`,
			http.StatusOK,
			`{
				"id": 99,
				"active": true,
				"anonymous": true,
				"comment": "a great comment",
				"date": 0,
				"extra": {"color": "red"},
				"score": 10,
				"target": 9999,
				"userId": 1
			}`,
			func(t *testing.T) {
				rs.update = func(mr *models.Rating) error {
					mr.User = nil
					mr.UserID = 1

					assert.Equal(t, &models.Rating{
						ID:        99,
						Active:    true,
						Anonymous: true,
						Comment:   "a great comment",
						Date:      0,
						Extra:     json.RawMessage(`{"color": "red"}`),
						Score:     10,
						Target:    9999,
						UserID:    1,
					}, mr)

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

			*rs = testRatingService{}
		})
	}
}

func TestRatings_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRatingService{}
	r := NewRatings(rs)

	mux := gin.New()
	mux.DELETE("/api/v1/ratings/:id", func(c *gin.Context) {
		c.Set("user", &models.User{
			ID: 1,
		})
	}, r.Delete)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badPathID",
			"/api/v1/ratings/lksdjflk",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/ratings/999",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				rs.delete = func(mr *models.Rating) error {
					assert.Equal(t, int64(999), mr.ID)
					return models.ErrNotFound
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/ratings/999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.delete = func(mr *models.Rating) error {
					assert.Equal(t, int64(999), mr.ID)
					return wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/ratings/999",
			http.StatusNoContent,
			`{}`,
			func(t *testing.T) {
				rs.delete = func(mr *models.Rating) error {
					assert.Equal(t, int64(999), mr.ID)
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

			*rs = testRatingService{}
		})
	}
}

func TestRatings_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRatingService{}
	r := NewRatings(rs)

	mux := gin.New()
	mux.GET("/api/v1/ratings/:id", r.Get)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badPathID",
			"/api/v1/ratings/lksdjflk",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/ratings/999",
			http.StatusNotFound,
			`{"error":"not_found"}`,
			func(t *testing.T) {
				rs.byID = func(id int64) (models.Rating, error) {
					assert.Equal(t, int64(999), id)
					return models.Rating{}, models.ErrNotFound
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/ratings/999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.byID = func(id int64) (models.Rating, error) {
					assert.Equal(t, int64(999), id)
					return models.Rating{}, wrap("test internal error", nil)
				}
			},
		},
		{
			"ok",
			"/api/v1/ratings/999",
			http.StatusOK,
			`{
				"id": 999,
				"active": true,
				"anonymous": true,
				"comment": "a great comment",
				"date": 0,
				"extra": {"color": "red"},
				"score": 10,
				"target": 9999,
				"userId": 1
			}`,
			func(t *testing.T) {
				rs.byID = func(id int64) (models.Rating, error) {
					assert.Equal(t, int64(999), id)

					return models.Rating{
						ID:        999,
						Active:    true,
						Anonymous: true,
						Comment:   "a great comment",
						Date:      0,
						Extra:     json.RawMessage(`{"color": "red"}`),
						Score:     10,
						Target:    9999,
						UserID:    1,
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

			*rs = testRatingService{}
		})
	}
}

func TestRatings_ListByTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRatingService{}
	r := NewRatings(rs)

	mux := gin.New()
	mux.GET("/api/v1/ratings/", r.ListByTarget)

	var cases = []struct {
		name      string
		path      string
		outStatus int
		outJSON   string
		setup     func(*testing.T)
	}{
		{
			"badQueryID",
			"/api/v1/ratings/?target=kdfjhgkhjlsfg",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target": "invalid_parse"}}`,
			nil,
		},
		{
			"badQueryID2",
			"/api/v1/ratings/?target=kdfjhg,khjlsfg",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target": "invalid_parse"}}`,
			nil,
		},
		{
			"badQueryID3",
			"/api/v1/ratings/?whatever=999",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target": "invalid_parse"}}`,
			nil,
		},
		{
			"blankTarget",
			"/api/v1/ratings/?target=",
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target": "invalid_parse"}}`,
			nil,
		},
		{
			"notInStore",
			"/api/v1/ratings/?target=999",
			http.StatusOK,
			`{"items":[]}`,
			func(t *testing.T) {
				rs.byTarget = func(id int64) ([]models.Rating, error) {
					assert.Equal(t, int64(999), id)
					return nil, nil
				}
			},
		},
		{
			"storeInternalError",
			"/api/v1/ratings/?target=999",
			http.StatusInternalServerError,
			`{"error":"server_error"}`,
			func(t *testing.T) {
				rs.byTarget = func(id int64) ([]models.Rating, error) {
					assert.Equal(t, int64(999), id)
					return nil, wrap("test internal error", nil)
				}
			},
		},
		{
			"okMultiple",
			"/api/v1/ratings/?target=99",
			http.StatusOK,
			`{"items":[
					{
						"id": 777,
						"active": true,
						"anonymous": true,
						"comment": "a great comment",
						"extra": {"color":"blue"},
						"date": 0,
						"score": 9,
						"target": 99,
						"userId": 8
					},
					{
						"id": 888,
						"active": true,
						"anonymous": true,
						"comment": "a great comment",
						"extra": {"color":"red"},
						"date": 0,
						"score": 10,
						"target": 99,
						"userId": 1
					}
				]}`,
			func(t *testing.T) {
				rs.byTarget = func(id int64) ([]models.Rating, error) {
					assert.Equal(t, int64(99), id)
					return []models.Rating{
						models.Rating{
							ID:        777,
							Active:    true,
							Anonymous: true,
							Comment:   "a great comment",
							Date:      0,
							Extra:     json.RawMessage(`{"color": "blue"}`),
							Score:     9,
							Target:    99,
							UserID:    8,
						},
						models.Rating{
							ID:        888,
							Active:    true,
							Anonymous: true,
							Comment:   "a great comment",
							Date:      0,
							Extra:     json.RawMessage(`{"color": "red"}`),
							Score:     10,
							Target:    99,
							UserID:    1,
						},
					}, nil
				}
			},
		},
		{
			"ok",
			"/api/v1/ratings/?target=99",
			http.StatusOK,
			`{"items":[
					{
						"id": 888,
						"active": true,
						"anonymous": true,
						"comment": "a great comment",
						"extra": {"color":"red"},
						"date": 0,
						"score": 10,
						"target": 99,
						"userId": 1
					}
				]}`,
			func(t *testing.T) {
				rs.byTarget = func(id int64) ([]models.Rating, error) {
					assert.Equal(t, int64(99), id)
					return []models.Rating{
						models.Rating{
							ID:        888,
							Active:    true,
							Anonymous: true,
							Comment:   "a great comment",
							Date:      0,
							Extra:     json.RawMessage(`{"color": "red"}`),
							Score:     10,
							Target:    99,
							UserID:    1,
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

			*rs = testRatingService{}
		})
	}
}
