package models

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"golang.org/x/xerrors"
)

// RatingService defines a set of methods to be used when dealing with ratings.
type RatingService interface {
	RatingDB
}

// RatingDB defines how the service interacts with the database.
type RatingDB interface {
	// Create adds a rating to the system. A target, score and userId are
	// required. The input parameter will be modified with normalised and
	// validated values and ID will be set to the new rating ID.
	//
	// Use NewRating() to use appropriate default values for the fields.
	//
	// Score field can be any number between -2,147,483,648 to 2,147,483,647.
	Create(*Rating) error

	// Update updates a rating in the system. A target, score and userId are
	// required. The input parameter will be modified with normalised
	// and validated values.
	//
	// Use NewRating() to use appropriate default values for the fields.
	//
	// The admin and user rating cannot be updated.
	Update(*Rating) error

	// Delete removes a rating by ID.
	Delete(int64) error

	// ByID retrieves a rating by ID.
	ByID(int64) (Rating, error)

	// ByTarget retrieves a list of ratings by their common target ID.
	ByTarget(int64) ([]Rating, error)
}

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
	UserID int64 `gorm:"unique_index:uix_ratings_user_id_target;type:bigint;not null" json:"userId"`
}

// NewRating creates a new Rating value with default field values applied.
func NewRating() Rating {
	return Rating{
		Active:    true,
		Anonymous: true,
		Extra:     json.RawMessage(`{}`),
	}
}

type ratingService struct {
	RatingService
}

// NewRatingService instantiates a new RatingService implementation with db as the
// backing database.
func NewRatingService(db *gorm.DB) RatingService {
	return &ratingService{
		RatingService: &ratingValidator{
			RatingDB: &ratingGorm{db},
		},
	}
}

type ratingValidator struct {
	RatingDB
}

func (rv *ratingValidator) Create(rating *Rating) error {
	err := rv.runValFuncs(rating,
		rv.idSetToZero,
		rv.setDate,
		rv.targetRequired,
		rv.scoreRequired,
		rv.userIDRequired,
		rv.commentLength,
		rv.extraLength,
		rv.targetInvalid,
		rv.userIDInvalid,
	)
	if err != nil {
		return err
	}

	return rv.RatingDB.Create(rating)
}

type ratingValFn func(r *Rating) error

func (rv *ratingValidator) runValFuncs(r *Rating, fns ...func() (string, ratingValFn)) error {
	return runValidationFunctions(r, fns)
}

// idSetToZero sets interface id to 0. It does not return any errors.
func (rv *ratingValidator) idSetToZero() (string, ratingValFn) {
	return "", func(r *Rating) error {
		r.ID = 0
		return nil
	}
}

// setDate sets date to now. It does not return any errors.
func (rv *ratingValidator) setDate() (string, ratingValFn) {
	return "", func(r *Rating) error {
		nowEpoch := time.Now().Unix()
		r.Date = nowEpoch
		return nil
	}
}

// targetRequired returns an error if the target is 0. It may return ErrRequired.
func (rv *ratingValidator) targetRequired() (string, ratingValFn) {
	return "target", func(r *Rating) error {
		if r.Target == 0 {
			return ErrRequired
		}
		return nil
	}
}

// targetInvalid returns an error if the target is less than 1. It may return
// ErrInvalid.
func (rv *ratingValidator) targetInvalid() (string, ratingValFn) {
	return "target", func(r *Rating) error {
		if r.Target < 1 {
			return ErrInvalid
		}
		return nil
	}
}

// scoreRequired returns an error if the score is 0. It may return ErrRequired.
func (rv *ratingValidator) scoreRequired() (string, ratingValFn) {
	return "score", func(r *Rating) error {
		if r.Score == 0 {
			return ErrRequired
		}
		return nil
	}
}

// userIDRequired returns an error if the userId is 0. It may return ErrRequired.
func (rv *ratingValidator) userIDRequired() (string, ratingValFn) {
	return "userId", func(r *Rating) error {
		if r.UserID == 0 {
			return ErrRequired
		}
		return nil
	}
}

// userIDRequired returns an error if the userId is 0. It may return ErrRequired.
func (rv *ratingValidator) userIDInvalid() (string, ratingValFn) {
	return "userId", func(r *Rating) error {
		if r.UserID < 1 {
			return ErrInvalid
		}
		return nil
	}
}

// commentLength makes sure the comment has a maximum of 512 characters.
// It may return ErrTooLong.
func (rv *ratingValidator) commentLength() (string, ratingValFn) {
	return "comment", func(r *Rating) error {
		if len(r.Comment) > 512 {
			return ErrTooLong
		}

		return nil
	}
}

// extraLength makes sure the extra has a maximum of 512 characters.
// It may return ErrTooLong.
func (rv *ratingValidator) extraLength() (string, ratingValFn) {
	return "extra", func(r *Rating) error {
		if len(r.Extra) > 512 {
			return ErrTooLong
		}

		return nil
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
