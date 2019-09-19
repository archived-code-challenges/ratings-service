package models

import (
	"encoding/json"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"golang.org/x/xerrors"
)

// The permission constants enumerate the permissions recognised by the application.
const (
	// PermissionReadUsers allows reading and listing user accounts.
	PermissionReadUsers Permissions = (1 << iota)

	// PermissionWriteUsers allows modifying and deleting user accounts.
	PermissionWriteUsers

	// PermissionReadRatings allow reading any ratings information.
	PermissionReadRatings

	// PermissionWriteRatings allow modifying ratings information.
	PermissionWriteRatings
)

var (
	permissionsFromString = map[string]Permissions{
		"readUsers":    PermissionReadUsers,
		"writeUsers":   PermissionWriteUsers,
		"readRatings":  PermissionReadRatings,
		"writeRatings": PermissionWriteRatings,
	}

	permissionsToString = map[Permissions]string{
		PermissionReadUsers:    "readUsers",
		PermissionWriteUsers:   "writeUsers",
		PermissionReadRatings:  "readRatings",
		PermissionWriteRatings: "writeRatings",
	}
)

// RoleService defines a set of methods to be used when dealing with system roles.
type RoleService interface {
	RoleDB
}

// RoleDB defines how the service interacts with the database.
type RoleDB interface {
	// Create adds a role to the system. The Label field is mandatory.
	// The input parameter will be modified with normalised and validated
	// values and ID will be set to the new role ID.
	//
	// Use NewRole() to use appropriate default values for the fields.
	//
	// Roles with labels "admin" and "user" cannot be created.
	Create(*Role) error

	// Update updates a role in the system. The Label and ID fields are
	// required. The input parameter will be modified with normalised
	// and validated values.
	//
	// Use NewRole() to use appropriate default values for the fields.
	//
	// The admin and user role cannot be updated.
	Update(*Role) error

	// Delete removes a role by ID. The admin and user roles with
	// IDs 1 and 2 cannot be removed.
	Delete(int64) error

	// ByID retrieves a role by ID.
	ByID(int64) (Role, error)

	// ByIDs retrieves a list of roles by their IDs. If
	// no ID is supplied, all roles in the database are returned.
	ByIDs(...int64) ([]Role, error)
}

// A Role gives a name to a set of permissions, and allows associating them to users.
type Role struct {
	ID int64 `gorm:"primary_key;type:bigserial" json:"id"`

	// Label uniquely identifies a role in the system. The
	// "admin" and "user" labels are system defaults and
	// cannot be used or modified.
	Label string `gorm:"unique;size:255;not null" json:"label"`

	// Permissions mark what actions are allowed to be executed
	// by users with this role.
	Permissions Permissions `gorm:"type:bigint;not null" json:"permissions"`
}

// NewRole creates a new Role value with default field values applied.
func NewRole() Role {
	return Role{}
}

// Permissions defines bitfields where each bit represents a set of operations a user is allowed to perform on the application.
//
// A set of constants determine what operations are available and accepted as permissions.
type Permissions int64

// UnmarshalJSON converts a JSON encoded array of strings into a Permission value.
func (p *Permissions) UnmarshalJSON(b []byte) error {
	var pl []string
	if err := json.Unmarshal(b, &pl); err != nil {
		return err
	}

	for _, e := range pl {
		if ps, ok := permissionsFromString[e]; ok {
			*p |= ps

		} else {
			return wrap("permission does not exist", nil)
		}
	}
	return nil
}

// MarshalJSON encodes p as a list of strings, each one representing an enumerated permission.
func (p Permissions) MarshalJSON() ([]byte, error) {
	var s = []string{}

	var max = uint64(0x8000000000000000)
	for i := Permissions(1); i != Permissions(max); i <<= 1 {
		if p&i != 0 {
			if sp, ok := permissionsToString[i]; ok {
				s = append(s, sp)
			}
		}
	}

	return json.Marshal(s)
}

type roleService struct {
	RoleService
}

// NewRoleService instantiates a new RoleService implementation with db as the
// backing database.
func NewRoleService(db *gorm.DB) RoleService {
	return &roleService{
		RoleService: &roleValidator{
			RoleDB: &roleGorm{db},
		},
	}
}

type roleValidator struct {
	RoleDB
}

func (rv *roleValidator) Create(role *Role) error {
	err := rv.runValFuncs(role,
		rv.idSetToZero,
		rv.labelRequired,
		rv.isNotAdmin,
		rv.isNotUser,
		rv.normaliseLabel,
		rv.labelLength,
	)
	if err != nil {
		return err
	}

	return rv.RoleDB.Create(role)
}

func (rv *roleValidator) Update(role *Role) error {
	err := rv.runValFuncs(role,
		rv.idNotAdmin,
		rv.idNotUser,
		rv.isNotAdmin,
		rv.isNotUser,
		rv.labelRequired,
		rv.normaliseLabel,
		rv.labelLength,
	)
	if err != nil {
		return err
	}
	return rv.RoleDB.Update(role)
}

func (rv *roleValidator) Delete(id int64) error {
	err := rv.runValFuncs(&Role{ID: id},
		rv.idNotAdmin,
		rv.idNotUser,
	)
	if err != nil {
		return err
	}

	return rv.RoleDB.Delete(id)
}

type roleValFn func(r *Role) error

func (rv *roleValidator) runValFuncs(r *Role, fns ...func() (string, roleValFn)) error {
	return runValidationFunctions(r, fns)
}

// idSetToZero sets interface id to 0. It does not return any errors.
func (rv *roleValidator) idSetToZero() (string, roleValFn) {
	return "", func(r *Role) error {
		r.ID = 0
		return nil
	}
}

// isNotAdmin makes sure the label is not equal to "admin". It may return ErrDuplicate.
func (rv *roleValidator) isNotAdmin() (string, roleValFn) {
	return "label", func(r *Role) error {
		if r.Label == "admin" {
			return ErrDuplicate
		}

		return nil
	}
}

// isNotUser makes sure the label is not equal to "user". It may return ErrDuplicate.
func (rv *roleValidator) isNotUser() (string, roleValFn) {
	return "label", func(r *Role) error {
		if r.Label == "user" {
			return ErrDuplicate
		}

		return nil
	}
}

// idNotAdmin makes sure the ID is not equal to 1. It may return ErrReadOnly.
func (rv *roleValidator) idNotAdmin() (string, roleValFn) {
	return "", func(r *Role) error {
		if r.ID == 1 {
			return ErrReadOnly
		}

		return nil
	}
}

// idNotUser makes sure the label is not equal to 2. It may return ErrReadOnly.
func (rv *roleValidator) idNotUser() (string, roleValFn) {
	return "", func(r *Role) error {
		if r.ID == 2 {
			return ErrReadOnly
		}

		return nil
	}
}

// labelRequired returns an error if the label is empty. It may return ErrRequired.
func (rv *roleValidator) labelRequired() (string, roleValFn) {
	return "label", func(r *Role) error {
		if r.Label == "" {
			return ErrRequired
		}
		return nil
	}
}

// normalizeLabel removes leading and trailing white space from the label. It
// does not return any errors.
func (rv *roleValidator) normaliseLabel() (string, roleValFn) {
	return "label", func(r *Role) error {
		r.Label = strings.ToLower(r.Label)
		r.Label = strings.TrimSpace(r.Label)
		r.Label = strings.Join(strings.Fields(r.Label), " ")
		return nil
	}
}

// labelLength makes sure the label has at least 4 characters. It may return ErrTooShort.
func (rv *roleValidator) labelLength() (string, roleValFn) {
	return "label", func(r *Role) error {
		if r.Label != "" && len(r.Label) < 4 {
			return ErrTooShort
		}

		return nil
	}
}

type roleGorm struct {
	db *gorm.DB
}

func (rg *roleGorm) Create(r *Role) error {
	res := rg.db.Create(r)

	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "roles_pkey":
				return ValidationError{"id": ErrIDTaken}
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "roles_label_key":
				return ValidationError{"label": ErrDuplicate}
			}
		}

		return wrap("could not create role", res.Error)
	}

	return nil
}

func (rg *roleGorm) Update(r *Role) error {
	res := rg.db.Model(&Role{ID: r.ID}).Updates(gormToMap(rg.db, r))

	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "roles_label_key":
				return ValidationError{"label": ErrDuplicate}
			}
		}

		return wrap("could not update role", res.Error)

	} else if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (rg *roleGorm) Delete(id int64) error {
	res := rg.db.Delete(&Role{}, id)

	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "foreign_key_violation":
				return ErrInUse
			}
		}
		return wrap("could not delete role", res.Error)

	} else if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (rg *roleGorm) ByID(id int64) (Role, error) {
	var role Role
	err := rg.db.First(&role, id).Error

	if err != nil {
		if xerrors.Is(err, gorm.ErrRecordNotFound) {
			return Role{}, ErrNotFound
		}
		return Role{}, wrap("could not get role by ID", err)
	}

	return role, err
}

func (rg *roleGorm) ByIDs(ids ...int64) ([]Role, error) {
	var roles []Role

	qb := rg.db
	if len(ids) > 0 {
		qb = qb.Where(ids)
	}

	err := qb.Find(&roles).Error
	if err != nil {
		return nil, wrap("failed to list roles by ids", err)
	}

	return roles, nil
}
