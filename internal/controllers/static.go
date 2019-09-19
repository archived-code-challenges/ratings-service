package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/views"
)

// The Static controller provides HTTP handlers for all static page serving needs of the
// application.
type Static struct {
	viewErr views.Error
}

// NewStatic initialises a Static controller.
func NewStatic() *Static {
	var ev views.Error
	ev.SetCode(ErrNotFound, http.StatusNotFound)

	return &Static{
		viewErr: ev,
	}
}

// NotFound either redirects to a Not Found error page or returns a JSON message and status code
// for routes that could not be resolved.
//
// If a request has an "Accept" header that contains "application/json", then the JSON message will
// be returned.
func (s *Static) NotFound(c *gin.Context) {
	// TODO: when working on serving the UI, this needs to decide between
	// returning JSON and returning a UI message.

	s.viewErr.JSON(c, ErrNotFound)
}

// Home redirects a user to the main UI page from the root route.
//
// GET /
func (s *Static) Home(c *gin.Context) {
	// TODO: this needs to be implemented, the home page
}
