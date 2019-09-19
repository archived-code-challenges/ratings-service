package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/noelruault/ratingsapp/internal/views"
	"golang.org/x/xerrors"
)

// Users implements a controller for authentication, authorisation and
// user management.
type Users struct {
	us models.UserService

	viewErr views.Error
}

// NewUsers creates a new Users controller.
func NewUsers(us models.UserService) *Users {
	var ev views.Error
	ev.SetCode(models.ErrDuplicate, http.StatusConflict)
	ev.SetCode(models.ErrFieldReadOnly, http.StatusConflict)
	ev.SetCode(models.ErrReadOnly, http.StatusConflict)
	ev.SetCode(ErrNotFound, http.StatusNotFound)
	ev.SetCode(models.ErrNotFound, http.StatusNotFound)

	return &Users{
		us:      us,
		viewErr: ev,
	}
}

// Login takes a username and password or a refresh token and returns a set of
// access and refresh tokens.
//
// Login takes care of its own Content-Types as it is not a standard API call. No
// middlewares for content types should be appliced to Login.
//
// POST /api/v1/oauth/token
func (u *Users) Login(c *gin.Context) {
	var auth struct {
		Email        string `form:"email"`
		Password     string `form:"password"`
		RefreshToken string `form:"refresh_token"`
		GrantType    string `form:"grant_type" binding:"required"` // password, client_credentials, refresh_token
	}

	// parse the form-encoded input
	err := parseForm(c, &auth)
	if err != nil {
		oauthBadRequest(c, err)
		return
	}

	// check grant-types
	var user models.User
	if auth.GrantType == "password" {
		user, err = u.us.Authenticate(auth.Email, auth.Password)
		if err != nil {
			oauthAuthError(c, err)
			return
		}

	} else if auth.GrantType == "refresh_token" {
		user, err = u.us.Refresh(auth.RefreshToken)
		if err != nil {
			oauthAuthError(c, err)
			return
		}

	} else {
		oauthBadGrantType(c)
		return
	}

	tok, err := u.us.Token(&user)
	if err != nil {
		oauthAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, &tok)
}

// Create adds a new user to the system.
//
// POST /api/v1/users/
func (u *Users) Create(c *gin.Context) {
	var user = models.NewUser()

	err := parseJSON(c, &user)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	err = u.us.Create(&user)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusCreated, &user)
}

// Update updates an existing user in the system.
//
// PUT /api/v1/users/:id
func (u *Users) Update(c *gin.Context) {
	id, err := getParamInt(c, "id")
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	var user = models.NewUser()

	err = parseJSON(c, &user)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}
	user.ID = id

	err = u.us.Update(&user)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusOK, &user)
}

// Delete removes a user by ID.
//
// DELETE /api/v1/users/:id
func (u *Users) Delete(c *gin.Context) {
	id, err := getParamInt(c, "id")
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	err = u.us.Delete(id)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// Get returns one user by ID to the requester.
//
// GET /api/v1/users/:id
func (u *Users) Get(c *gin.Context) {
	id, err := getParamInt(c, "id")
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	user, err := u.us.ByID(id)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusOK, &user)
}

// List returns a list of users, optionally filteres by IDs, to the requester.
//
// The IDs are passed as a comma-separated list of user IDs, as the "id" query parameter.
// If any ID passed are not found, those are not shown on the returned list.
//
// This handler will never return a NotFound error, instead returnind an empty list.
//
// GET /api/v1/users/?id=1,2,3
func (u *Users) List(c *gin.Context) {
	ids, err := getQueryListInt(c, "id")
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	users, err := u.us.ByIDs(ids...)
	if err != nil {
		u.viewErr.JSON(c, err)
		return
	}

	if users == nil {
		users = []models.User{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": users,
	})
}

func oauthBadRequest(c *gin.Context, err error) {
	out := gin.H{
		"error": "invalid_request",
	}

	if pe, ok := err.(publicError); ok {
		out["error_description"] = pe.Public()
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, out)
}

func oauthAuthError(c *gin.Context, err error) {
	if xerrors.Is(err, models.ErrUnauthorised) {

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_client",
		})
		return

	} else if pe, ok := err.(publicError); ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": pe.Public(),
		})
		return
	}

	c.Error(err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": "server_error",
	})
}

func oauthBadGrantType(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"error": "unsupported_grant_type",
	})
}
