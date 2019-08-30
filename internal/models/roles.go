package models

import (
	"encoding/json"

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
