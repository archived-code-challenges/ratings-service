package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/noelruault/ratingsapp/internal/views"
)

// Ratings implements a controller for rating management.
type Ratings struct {
	rs models.RatingService

	viewErr views.Error
}

// NewRatings creates a new Ratings controller.
func NewRatings(rs models.RatingService) *Ratings {
	var ev views.Error
	ev.SetCode(ErrNotFound, http.StatusNotFound)
	ev.SetCode(models.ErrNotFound, http.StatusNotFound)
	ev.SetCode(models.ErrRefNotFound, http.StatusNotFound)

	return &Ratings{
		rs:      rs,
		viewErr: ev,
	}
}

// Create performs the addition of a rating.
//
// POST /api/v1/ratings/
func (r *Ratings) Create(c *gin.Context) {

	// user will be used to attach the rating to a specific user.
	user, exists := c.Get("user")
	if !exists {
		r.viewErr.JSON(c, errors.New("user couldn't be get from the context"))
		return
	}

	var rating = models.NewRating()

	u, ok := user.(*models.User)
	if !ok {
		panic("user from the context must be a pointer of models.User type")
	}

	err := parseJSON(c, &rating)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	rating.User = u

	err = r.rs.Create(&rating)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusCreated, &rating)
}

// Update performs the alteration of a rating in the system.
//
// PUT /api/v1/ratings/:id
func (r *Ratings) Update(c *gin.Context) {

	id, err := getParamInt(c, "id")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	user, exists := c.Get("user")
	if !exists {
		r.viewErr.JSON(c, errors.New("user couldn't be obtained from the context"))
		return
	}

	var rating = models.NewRating()

	u, ok := user.(*models.User)
	if !ok {
		panic("user from the context must be a pointer of models.User type")
	}

	err = parseJSON(c, &rating)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	rating.ID = id
	rating.User = u

	err = r.rs.Update(&rating)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusOK, &rating)
}

// Delete performs the removal of a rating.
//
// DELETE /api/v1/ratings/:id
func (r *Ratings) Delete(c *gin.Context) {

	id, err := getParamInt(c, "id")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	err = r.rs.Delete(id)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// Get returns one rating by ID to the requester.
//
// GET /api/v1/ratings/:id
func (r *Ratings) Get(c *gin.Context) {
	id, err := getParamInt(c, "id")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	rating, err := r.rs.ByID(id)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusOK, &rating)
}

// ListByTarget returns a list of ratings for a given target
//
// GET /api/v1/ratings/?target=999
func (r *Ratings) ListByTarget(c *gin.Context) {
	tid, err := getQueryParam(c, "target")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	ratings, err := r.rs.ByTarget(tid)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	if ratings == nil {
		ratings = []models.Rating{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": ratings,
	})
}
