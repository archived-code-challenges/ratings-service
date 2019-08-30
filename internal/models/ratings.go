package models

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"golang.org/x/xerrors"
)

// A Rating represents a valoration in the system from a user to an object.
type Rating struct {
	ID int64 `gorm:"primary_key;type:bigserial" json:"id"`

	// Whether the rate is active.
	Active bool `gorm:"not null" json:"active"`

	// Whether the rating is anonymous or not.
	Anonymous bool `gorm:"not null" json:"anonymous"`

	// The commentary attached to the rating.
	Comment string `gorm:"type:text;not null" json:"comment,omitempty"`

	// Date when the rating was submitted or updated. Any input date will be
	// ignored.
	Date int64 `gorm:"type:bigint;not null" json:"date"`

	// field to store stuff like logistics, color, date... in a json format.
	Extra json.RawMessage `gorm:"not null"  json:"extra"`

	// Numeral value that will indicate the score that the target got in a rating.
	// Score needs to be different than 0, but can be a negative value.
	Score int `gorm:"type:int;not null" json:"score"`

	// Numeral value that contains the target entity of the rating.
	Target int64 `gorm:"unique_index:uix_ratings_user_id_target;type:;bigint;not null" json:"target"`

	// The ID of the user attached to this rating.
	UserID int64 `gorm:"unique_index:uix_ratings_user_id_target;type:bigint;not null" json:"userId"` // REMOVEME
	// User User `gorm:"unique_index; json:"user,omitempty"`
}

// NewRating creates a new Rating value with default field values applied.
func NewRating() Rating {
	return Rating{
		Active:    true,
		Anonymous: true,
		Extra:     json.RawMessage(`{}`),
	}
}

type ratingGorm struct {
	db *gorm.DB
}

func (rg *ratingGorm) Create(r *Rating) error {
	res := rg.db.Create(r)

	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "ratings_pkey":
				return ValidationError{"id": ErrIDTaken}
			case perr.Code.Name() == "foreign_key_violation" && perr.Constraint == "ratings_user_id_users_id_foreign":
				return ValidationError{"userId": ErrRefNotFound}
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "uix_ratings_user_id_target":
				return ValidationError{"target": ErrDuplicate}
			}
		}

		return wrap("could not create rating", res.Error)
	}

	return nil
}

func (rg *ratingGorm) Update(r *Rating) error {
	res := rg.db.Model(&Rating{ID: r.ID}).Updates(gormToMap(rg.db, r))

	if res.Error != nil {
		if perr := (*pq.Error)(nil); xerrors.As(res.Error, &perr) {
			switch {
			case perr.Code.Name() == "foreign_key_violation" && perr.Constraint == "ratings_user_id_users_id_foreign":
				return ValidationError{"userId": ErrRefNotFound}
			case perr.Code.Name() == "unique_violation" && perr.Constraint == "uix_ratings_user_id_target":
				return ValidationError{"target": ErrDuplicate}
			}
		}

		return wrap("could not update rating", res.Error)

	} else if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (rg *ratingGorm) Delete(id int64) error {
	res := rg.db.Delete(&Rating{}, id)

	if res.Error != nil {
		return wrap("could not delete rating by id", res.Error)

	} else if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (rg *ratingGorm) ByID(id int64) (Rating, error) {
	var rating Rating
	err := rg.db.First(&rating, id).Error

	if err != nil {
		if xerrors.Is(err, gorm.ErrRecordNotFound) {
			return Rating{}, ErrNotFound
		}
		return Rating{}, wrap("could not get rating by ID", err)
	}

	return rating, err
}

func (rg *ratingGorm) ByTarget(target int64) ([]Rating, error) {
	var ratings []Rating
	err := rg.db.Where("target = ?", target).Find(&ratings).Error

	if err != nil {
		if xerrors.Is(err, gorm.ErrRecordNotFound) {
			return []Rating{}, nil
		}
		return []Rating{}, wrap("failed to list ratings by target", err)
	}

	return ratings, nil
}
