package models

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"golang.org/x/xerrors"
)

const (
	// waitAfterAuthError is the period to sleep after a failed user authentication attempt.
	waitAfterAuthError = 500 * time.Millisecond

	jwtAccessDuration  = 6 * time.Hour
	jwtRefreshDuration = 10 * 24 * time.Hour
)

// A User represents an application user, be it a human or another application
// that connects to this one.
type User struct {
	ID int64 `gorm:"primary_key;type:bigserial" json:"id"`

	// Active marks if the user is active in the system or
	// disabled. Inactive users are not able to login or
	// use the system.
	Active bool `gorm:"not null" json:"active"`

	// Email is the actual user identifier in the system
	// and must be unique.
	Email string `gorm:"unique;size:255;not null" json:"email"`

	// FirstName is the user's first name or an application user's
	// description.
	FirstName string `gorm:"size:255;not null" json:"firstName"`

	// LastName is the user's last name or last names, and it may
	// be left blank.
	LastName string `gorm:"size:255;not null" json:"lastName"`

	// Password stores the hashed user's password. This value
	// is always cleared when the services return a new user.
	Password string `gorm:"size:255;not null" json:"password,omitempty"`

	// RoleID points to the role this user is attached to. A
	// role defines what a user is able to do in the system.
	RoleID int64 `gorm:"type:bigint;not null" json:"roleId"`

	// Role contains the role pointed by RoleID. It may or may not be included
	// by the UserService methods.
	Role *Role `json:"role,omitempty"`

	// Settings is used by the frontend to store free-form
	// contents related to user preferences.
	Settings string `gorm:"type:text;not null" json:"settings,omitempty"` // settings information related to a user.
}

type userGorm struct {
	db *gorm.DB
}

func (ug *userGorm) Create(u *User) error {
	res := ug.db.Create(u)
	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "users_pkey":
				return ValidationError{"id": ErrIDTaken}
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "users_email_key":
				return ValidationError{"email": ErrDuplicate}
			case perr.Code.Name() == "foreign_key_violation" && perr.Constraint == "users_role_id_roles_id_foreign":
				return ValidationError{"roleId": ErrRefNotFound}
			}
		}

		return wrap("could not create user", res.Error)
	}

	return nil
}

func (ug *userGorm) Update(u *User) error {
	res := ug.db.Model(&User{ID: u.ID}).Updates(gormToMap(ug.db, u))

	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "users_email_key":
				return ValidationError{"email": ErrDuplicate}
			case perr.Code.Name() == "foreign_key_violation" && perr.Constraint == "users_role_id_roles_id_foreign":
				return ValidationError{"roleId": ErrRefNotFound}
			}
		}

		return wrap("could not update user", res.Error)

	} else if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (ug *userGorm) Delete(id int64) error {
	res := ug.db.Delete(&User{}, id)
	if res.Error != nil {
		return wrap("could not delete user by id", res.Error)

	} else if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (ug *userGorm) ByEmail(e string) (User, error) {
	var user User

	err := ug.db.Where("email = ?", e).Preload("Role").First(&user).Error
	if err != nil {
		if xerrors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		}

		return User{}, wrap("could not get user by email", err)
	}

	return user, nil
}

func (ug *userGorm) ByID(id int64) (User, error) {
	var user User

	err := ug.db.Preload("Role").First(&user, id).Error
	if err != nil {
		if xerrors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		}

		return User{}, wrap("could not get user by id", err)
	}

	return user, nil
}

func (ug *userGorm) ByIDs(ids ...int64) ([]User, error) {
	var users []User

	qb := ug.db
	if len(ids) > 0 {
		qb = qb.Where(ids)
	}

	err := qb.Find(&users).Error
	if err != nil {
		return nil, wrap("failed to list users by ids", err)
	}

	return users, nil
}
