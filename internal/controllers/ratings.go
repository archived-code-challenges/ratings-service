package controllers

import (
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

	return &Ratings{
		rs:      rs,
		viewErr: ev,
	}
}

// Create performs the addition of a rating.
//
// POST /api/v1/ratings/
func (r *Ratings) Create(c *gin.Context) {
	var rating = models.NewRating()

	err := parseJSON(c, &rating)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	err = r.rs.Create(&rating)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusCreated, &rating)
}
