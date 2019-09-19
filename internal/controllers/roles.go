package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/noelruault/ratingsapp/internal/views"
)

// Roles implements a controller for role management.
type Roles struct {
	rs models.RoleService

	viewErr views.Error
}

// NewRoles creates a new Roles controller.
func NewRoles(rs models.RoleService) *Roles {
	var ev views.Error
	ev.SetCode(models.ErrDuplicate, http.StatusConflict)
	ev.SetCode(models.ErrFieldReadOnly, http.StatusConflict)
	ev.SetCode(models.ErrReadOnly, http.StatusConflict)
	ev.SetCode(ErrNotFound, http.StatusNotFound)
	ev.SetCode(models.ErrNotFound, http.StatusNotFound)

	return &Roles{
		rs:      rs,
		viewErr: ev,
	}
}

// Create performs the addition of a role.
//
// POST /api/v1/roles/
func (r *Roles) Create(c *gin.Context) {
	var role = models.NewRole()

	err := parseJSON(c, &role)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	err = r.rs.Create(&role)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusCreated, &role)
}

// Update performs the change of a role.
//
// PUT /api/v1/roles/:id
func (r *Roles) Update(c *gin.Context) {
	id, err := getParamInt(c, "id")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	var role = models.NewRole()

	err = parseJSON(c, &role)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}
	role.ID = id

	err = r.rs.Update(&role)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusOK, &role)
}

// Delete performs the removal of a role.
//
// DELETE /api/v1/roles/:id
func (r *Roles) Delete(c *gin.Context) {
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

// Get returns one role by ID to the requester.
//
// GET /api/v1/roles/:id
func (r *Roles) Get(c *gin.Context) {
	id, err := getParamInt(c, "id")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	role, err := r.rs.ByID(id)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	c.JSON(http.StatusOK, &role)
}

// List returns a list of roles, optionally filters by IDs.
//
// The IDs are passed as a comma-separated list of role IDs, as the "id" query parameter.
// If any ID passed are not found, those are not shown on the returned list.
//
// This handler will never return a NotFound error, instead returnind an empty list.
//
// GET /api/v1/roles/?id=1,2,3
func (r *Roles) List(c *gin.Context) {
	ids, err := getQueryListInt(c, "id")
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	roles, err := r.rs.ByIDs(ids...)
	if err != nil {
		r.viewErr.JSON(c, err)
		return
	}

	if roles == nil {
		roles = []models.Role{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": roles,
	})
}
