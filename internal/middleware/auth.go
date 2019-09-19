package middleware

import (
	"net/http"

	"github.com/noelruault/ratingsapp/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/views"
)

// viewErr is a global Error view to be used by middleware when returning errors
// to users.
var viewErr = func() views.Error {
	var ev views.Error
	ev.SetCode(models.ErrUnauthorised, http.StatusUnauthorized)
	ev.SetCode(ErrForbidden, http.StatusForbidden)
	ev.SetCode(ErrNotAcceptable, http.StatusNotAcceptable)

	return ev
}()

// UserService is a subset of the models.UserService interface, containing only
// the methods required to run middleware.
type UserService interface {
	Validate(string) (models.User, error)
}

// Authenticated is a middleware that will only allow a request to go through if
// a user is authenticated. A "user" value of type *User is set on successfully
// authenticated contexts.
//
// The authentication is verified by checking a token passed in an HTTP header
// in the request. If authentication fails, an HTTP Unauthorized error is
// returned with a JSON description.
func Authenticated(us UserService) gin.HandlerFunc {

	return func(c *gin.Context) {
		tok := c.GetHeader("Authorization")
		if len(tok) < 8 || tok[:7] != "Bearer " {
			viewErr.JSON(c, models.ErrUnauthorised)
			return
		}

		user, err := us.Validate(tok[7:])
		if err != nil {
			viewErr.JSON(c, err)
			return
		}

		c.Set("user", &user)
		c.Next()
	}
}

// Can is a decorator for Gin handlers that verifies user permissions before h
// is executed. In case a user does not have enough permissions, a Forbidden
// message is be returned.
//
// If p has more than one permission listed, Can will allow the request if the
// user's role permits ALL of the roles in p.
func Can(p models.Permissions, h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		if user.Role.Permissions&p != p {
			viewErr.JSON(c, ErrForbidden)
			return
		}

		h(c)
	}
}
