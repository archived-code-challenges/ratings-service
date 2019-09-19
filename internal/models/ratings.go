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
	Delete(*Rating) error

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
	Extra json.RawMessage `gorm:"not null" json:"extra"`

	// Numeral value that will indicate the score that the target got in a rating.
	// Score needs to be different than 0, but can be a negative value.
	Score int `gorm:"type:int;not null" json:"score"`

	// Numeral value that contains the target entity of the rating.
	Target int64 `gorm:"unique_index:uix_ratings_user_id_target;type:;bigint;not null" json:"target"`

	// The ID of the user attached to this rating.
	UserID int64 `gorm:"unique_index:uix_ratings_user_id_target;type:bigint;not null" json:"userId"`

	// User contains the data that belongs to the user making the request.
	User *User `gorm:"-" json:"-"`
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
func NewRatingService(db *gorm.DB, us UserService) RatingService {
	return &ratingService{
		RatingService: &ratingValidator{
			RatingDB:    &ratingGorm{db},
			userService: us,
		},
	}
}

type ratingValidator struct {
	RatingDB
	userService UserService
}

func (rv *ratingValidator) Create(rating *Rating) error {
	rc := ratingValWithDBData{rv: rv, us: rv.userService}
	err := rv.runValFuncs(rating,
		rv.idSetToZero,
		rv.userSessionExists,
		rv.userSessionInvalid,
		rv.targetRequired,
		rv.scoreRequired,
		rv.commentLength,
		rv.extraLength,
		rv.targetInvalid,
		rc.fetchUser,
		rc.setSessionUserAsUserID,
		rv.setDate,
	)
	if err != nil {
		return err
	}

	rating.User = nil
	return rv.RatingDB.Create(rating)
}

func (rv *ratingValidator) Update(rating *Rating) error {
	rc := ratingValWithDBData{rv: rv, us: rv.userService}
	err := rv.runValFuncs(rating,
		rv.userSessionExists,
		rv.userSessionInvalid,
		rv.scoreRequired,
		rv.commentLength,
		rv.extraLength,
		rc.fetchUser,
		rc.fetchRating,
		rc.userIsOwner,
		rc.setDatabaseRatingDefaults,
		rc.setSessionUserAsUserID,
		rv.setDate,
	)
	if err != nil {
		return err
	}

	rating.User = nil
	return rv.RatingDB.Update(rating)
}

func (rv *ratingValidator) Delete(rating *Rating) error {

	rc := ratingValWithDBData{rv: rv, us: rv.userService}
	err := rv.runValFuncs(rating,
		rv.userSessionExists,
		rv.userSessionInvalid,
		rc.fetchUser,
		rc.fetchRating,
		rc.userIsOwnerOrAdmin,
	)

	if err != nil {
		return err
	}

	rating.User = nil
	return rv.RatingDB.Delete(rating)
}

type ratingValFn func(r *Rating) error

type ratingValWithDBData struct {
	rv          *ratingValidator
	us          UserService
	sessionUser User
	dbRating    Rating
}

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

// userSessionExists returns an error if the rating. User is nil. It may return
// an internal error with a descriptive message and ErrRequired attached.
func (rv *ratingValidator) userSessionExists() (string, ratingValFn) {
	return "", func(r *Rating) error {
		if r.User == nil {
			return wrap("session user does not exist", ErrRequired)
		}
		return nil
	}
}

// userSessionInvalid returns an error if the User ID is 0. It may return
// an internal error with a descriptive message and ErrInvalid attached.
// Must be called after userSessionExists.
func (rv *ratingValidator) userSessionInvalid() (string, ratingValFn) {
	return "", func(r *Rating) error {
		if r.User.ID < 1 {
			return wrap("session user is invalid", ErrInvalid)
		}
		return nil
	}
}

// fetchUser retrieves the current user value from the database. Must be called
// just after userSessionExists and userSessionInvalid which will check that the
// user on the session exists and can be used. Should be called just before any
// other validators implemented by the receiver type.
func (rc *ratingValWithDBData) fetchUser() (string, ratingValFn) {
	return "", func(r *Rating) error {
		var err error
		rc.sessionUser, err = rc.us.ByID(r.User.ID) // r.User must be in the context before being validated.
		if err != nil {
			return err
		}
		return nil
	}
}

// fetchRating retrieves the current rating value from the database. Must be
// called just after userSessionExists and userSessionInvalid which will check
// that the user on the session exists and can be used. Should be called just
// before any other validators implemented by the receiver type.
func (rc *ratingValWithDBData) fetchRating() (string, ratingValFn) {
	return "", func(r *Rating) error {
		var err error
		rc.dbRating, err = rc.rv.RatingDB.ByID(r.ID)
		if err != nil {
			return err
		}
		return nil
	}
}

// userIsOwner checks if the user of the session is the owner of the rating
// being processed. A session user must have been successfully obtained before
// using this method.
func (rc *ratingValWithDBData) userIsOwner() (string, ratingValFn) {
	return "", func(r *Rating) error {

		if rc.dbRating.UserID != rc.sessionUser.ID {
			return ErrReadOnly
		}
		return nil
	}
}

// setDatabaseRatingDefaults sets the target of the rating being processed to
// its existing value in the database. This method is dependent on fetchRating.
func (rc *ratingValWithDBData) setDatabaseRatingDefaults() (string, ratingValFn) {
	return "", func(r *Rating) error {
		r.Target = rc.dbRating.Target
		return nil
	}
}

// setDatabaseRatingDefaults sets the UserID of the rating being processed to
// the existing value of user retrieved from the session. This method is then
// dependent on fetchUser.
func (rc *ratingValWithDBData) setSessionUserAsUserID() (string, ratingValFn) {
	return "", func(r *Rating) error {
		r.UserID = rc.sessionUser.ID
		return nil
	}
}

// userIsOwnerOrAdmin proves that the user trying to delete a rating is the
// owner or an administrator. A session user must have been successfully
// obtained before using this method.
func (rc *ratingValWithDBData) userIsOwnerOrAdmin() (string, ratingValFn) {
	return "", func(r *Rating) error {

		if rc.dbRating.UserID != rc.sessionUser.ID {
			if r.User.RoleID != 1 {
				return ErrReadOnly
			}
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

func (rg *ratingGorm) Delete(r *Rating) error {
	res := rg.db.Delete(&Rating{}, r.ID)

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
