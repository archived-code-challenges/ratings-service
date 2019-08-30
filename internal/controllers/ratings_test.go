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

type testRatingService struct {
	models.RatingService
	create func(*models.Rating) error
}

func (t *testRatingService) Create(mr *models.Rating) error {
	if t.create != nil {
		return t.create(mr)
	}

	panic("not provided")
}

func TestRatings_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rs := &testRatingService{}
	r := NewRatings(rs)

	mux := gin.New()
	mux.POST("/api/v1/ratings/", r.Create)

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
				target: asldfkj,
				userId: asldfkj
			}`,
			http.StatusBadRequest,
			`{"error":"invalid_json"}`,
			nil,
		},
		{
			"internalError",
			`{
				"score": 10,
				"target": 9999,
				"userId": 1
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
				"target": -9999,
				"userId": 1
			}`,
			http.StatusBadRequest,
			`{"error":"validation_error","fields":{"target":"invalid"}}`,
			func(t *testing.T) {
				rs.create = func(r *models.Rating) error {
					rating := models.NewRating()
					rating.Score = 10
					rating.Target = -9999
					rating.UserID = 1

					assert.Equal(t, &rating, r)
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
				"target": 9999,
				"userId": 1
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
