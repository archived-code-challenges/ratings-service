package models

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/xerrors"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	// waitAfterAuthError is the period to sleep after a failed user authentication attempt.
	waitAfterAuthError = 500 * time.Millisecond

	jwtAccessDuration  = 6 * time.Hour
	jwtRefreshDuration = 10 * 24 * time.Hour
)

// UserService defines a set of methods to be used when dealing with system users
// and authenticating them.
type UserService interface {
	// Authenticate returns a user based on provided username and password.
	//
	// Errors returned include ErrNoCredentials and ErrUnauthorised. Specific
	// validation errors are masked and not provided, being replaced by
	// ErrUnauthorised.
	Authenticate(username, password string) (User, error)

	// Refresh returns a user based on a valid refresh token.
	Refresh(refreshToken string) (User, error)

	// Validate returns a user based on a valid access token.
	Validate(accessToken string) (User, error)

	// Token generates a set of tokens based on the user provided as
	// input.
	Token(u *User) (Token, error)

	UserDB
}

// UserDB defines how the service interacts with the database.
type UserDB interface {
	// Create adds a user to the system. For common users, the
	// Email, FirstName and Password are mandatory. For
	// application users, only FirstName and RoleID are mandatory.
	// The parameter u will be modified with normalised and validated
	// values and ID will be set to the new user ID.
	//
	// Use NewUser() to use appropriate default values for the other
	// fields.
	//
	// For application users, email and password will be generated.
	Create(u *User) error

	// Update updates a user in the system. For common users, the
	// Email, FirstName and Password are mandatory. For
	// application users, only FirstName and RoleID are mandatory.
	// The parameter u will be modified with normalised and validated
	// values.
	//
	// Use NewUser() to use appropriate default values for the other
	// fields.
	//
	// For application users, email and password cannot be updated.
	Update(u *User) error

	// Delete removes a user by ID. The admin user with ID
	// 1 cannot be removed.
	Delete(int64) error

	// ByID retrieves a user by ID.
	ByID(int64) (User, error)

	// ByIDs retrieves a list of users by their IDs. If
	// no ID is supplied, all users in the database are returned.
	ByIDs(...int64) ([]User, error)

	// ByEmail retrieves a user by email address, as it
	// is unique in the database.
	ByEmail(string) (User, error)
}

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

// NewUser creates a new User value with default field values applied.
func NewUser() User {
	return User{
		RoleID: 2,
		Active: true,
	}
}

// A Token is a set of tokens that represent a user logged in the system.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// ValidationResult contains the result of a token validation request.
type ValidationResult struct {
	UserID    int64
	RoleID    int64
	IsRefresh bool
}

type authClaims struct {
	jwt.Claims
	RoleID int64 `json:"fdr,omitempty"`
}

type userService struct {
	UserService

	signer jose.Signer
	secret []byte
}

// NewUserService instantiates a new UserService implementation with db as the
// backing database.
func NewUserService(db *gorm.DB, rs RoleService, jwtSecret []byte) (UserService, error) {
	sig, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS512,
		Key:       []byte(jwtSecret),
	}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return nil, wrap("failed to instantiate JWT signer", err)
	}

	return &userService{
		UserService: &userValidator{
			UserDB:      &userGorm{db},
			roleService: rs,
			emailRegex:  regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9._\-]+\.[a-z0-9._\-]{2,16}$`),
		},
		signer: sig,
		secret: jwtSecret,
	}, nil
}

func (us *userService) Authenticate(username, password string) (User, error) {
	// hide the actual errors to reduce ease of BF attacks.
	user, err := us.UserService.Authenticate(username, password)
	if err != nil {
		if xerrors.Is(err, ValidationError{"email": ErrRequired}) ||
			xerrors.Is(err, ValidationError{"password": ErrRequired}) {
			return user, ErrNoCredentials

		} else if verr := ValidationError(nil); xerrors.As(err, &verr) {
			if verr["password"] == ErrPasswordIncorrect {
				return user, ErrUnauthorised
			}

			err = ErrUnauthorised

		} else if merr := ModelError(""); xerrors.As(err, &merr) {
			err = ErrUnauthorised
		}

		// protection sleep to reduce effectiveness of BF attacks
		time.Sleep(waitAfterAuthError)
		return User{}, err
	}

	return user, nil
}

func (us *userService) Refresh(refreshToken string) (User, error) {
	if refreshToken == "" {
		return User{}, ErrNoCredentials
	}

	// validate the token
	uid, _, err := us.tokenValidate(refreshToken, true)
	if err != nil {
		if merr := ModelError(""); xerrors.As(err, &merr) {
			return User{}, ErrUnauthorised
		}

		return User{}, wrap("failed to validate refresh token", err)
	}

	// get the user from the database
	user, err := us.ByID(uid)
	if err != nil {
		if xerrors.Is(err, ErrNotFound) {
			return User{}, ErrUnauthorised
		}

		return User{}, wrap("on refresh, failed to obtain user from database", err)
	}

	if !user.Active {
		return User{}, ErrUnauthorised
	}

	return user, nil
}

func (us *userService) Validate(accessToken string) (User, error) {
	if accessToken == "" {
		return User{}, ErrUnauthorised
	}

	// validate the token
	uid, _, err := us.tokenValidate(accessToken, false)
	if err != nil {
		if merr := ModelError(""); xerrors.As(err, &merr) {
			return User{}, ErrUnauthorised
		}

		return User{}, wrap("failed to validate refresh token", err)
	}

	// get the user from the database
	user, err := us.ByID(uid)
	if err != nil {
		if xerrors.Is(err, ErrNotFound) {
			return User{}, ErrUnauthorised
		}

		return User{}, wrap("on validate, failed to obtain user from database", err)
	}

	if !user.Active {
		return User{}, ErrUnauthorised
	}

	return user, nil
}

func (us *userService) Token(u *User) (Token, error) {
	cla := authClaims{
		Claims: jwt.Claims{
			Subject: strconv.FormatInt(u.ID, 10),
			Issuer:  "ratingsapp",
			Expiry:  jwt.NewNumericDate(time.Now().UTC().Add(jwtAccessDuration)),
		},
		RoleID: u.RoleID,
	}
	clr := authClaims{
		Claims: jwt.Claims{
			Subject: strconv.FormatInt(u.ID, 10),
			Issuer:  "ratingsappr",
			Expiry:  jwt.NewNumericDate(time.Now().UTC().Add(jwtRefreshDuration)),
		},
		RoleID: u.RoleID,
	}

	atok, err := jwt.Signed(us.signer).Claims(cla).CompactSerialize()
	if err != nil {
		return Token{}, wrap("failed to generate access token", err)
	}

	rtok, err := jwt.Signed(us.signer).Claims(clr).CompactSerialize()
	if err != nil {
		return Token{}, wrap("failed to generate refresh token", err)
	}

	return Token{
		AccessToken:  atok,
		RefreshToken: rtok,
		ExpiresIn:    int(jwtAccessDuration / time.Second),
		TokenType:    "bearer",
	}, nil
}

func (us *userService) ByID(id int64) (User, error) {
	u, err := us.UserService.ByID(id)

	u.Password = ""
	return u, err
}

func (us *userService) ByIDs(ids ...int64) ([]User, error) {
	u, err := us.UserService.ByIDs(ids...)

	for i := range u {
		u[i].Password = ""
	}

	return u, err
}

func (us *userService) ByEmail(e string) (User, error) {
	u, err := us.UserService.ByEmail(e)

	u.Password = ""
	return u, err
}

// tokenValidate validates token as a JWT. If refresh is true, it validates it as being a
// refresh token. The method returns the user id and role id present in the token claims
func (us *userService) tokenValidate(token string, isRefresh bool) (uid, rid int64, err error) {
	var cl = authClaims{}

	// parse the token first
	tok, err := jwt.ParseSigned(token)
	if err != nil {
		return 0, 0, ErrRefreshInvalid
	}

	// verify the claims check with the signature key
	err = tok.Claims(us.secret, &cl)
	if err != nil {
		return 0, 0, ErrRefreshInvalid
	}

	// verify the token has not expired
	iss := "ratingsapp"
	if isRefresh {
		iss = "ratingsappr"
	}

	err = cl.Validate(jwt.Expected{
		Issuer: iss,
		Time:   time.Now().UTC(),
	})
	if err != nil {
		if xerrors.Is(err, jwt.ErrExpired) {
			return 0, 0, ErrRefreshExpired
		}

		return 0, 0, ErrRefreshInvalid
	}

	// get the user ID in the claim, passed in the subject field
	id, err := strconv.ParseInt(cl.Subject, 10, 0)
	if err != nil {
		return 0, 0, ErrRefreshInvalid
	}

	return id, cl.RoleID, nil
}

type userValidator struct {
	UserDB
	roleService RoleService
	emailRegex  *regexp.Regexp
}

func (uv *userValidator) Authenticate(username, password string) (User, error) {
	// create a simple user to apply validators
	user := User{
		Email:    username,
		Password: password,
	}

	err := uv.runValFuncs(&user,
		uv.emailRequired,
		uv.passwordRequired,
		uv.passwordLength,
		uv.normaliseEmail,
		uv.emailFormat,
	)
	if err != nil {
		return User{}, err
	}

	// fetch real user from DB after basic validation passes
	user, err = uv.UserDB.ByEmail(user.Email)
	if err != nil {
		return User{}, err
	}

	if !user.Active {
		return User{}, ErrInvalid
	}

	// check the password matches
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		if xerrors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return User{}, ValidationError{"password": ErrPasswordIncorrect}
		}

		return User{}, wrap("failed to compare password hashes", err)
	}

	return user, nil
}

func (uv *userValidator) Refresh(refreshToken string) (User, error) {
	panic("method Refresh of userValidator must never be called")
}

func (uv *userValidator) Validate(accessToken string) (User, error) {
	panic("method Validate of userValidator must never be called")
}

func (uv *userValidator) Token(u *User) (Token, error) {
	panic("method Token of userValidator must never be called")
}

func (uv *userValidator) Create(u *User) error {
	var pw string
	defer func() {
		u.Password = pw
	}()

	if err := uv.runValFuncs(u,
		uv.idSetToZero,
		uv.firstNameRequired,
		uv.firstNameLength,
		uv.settingsLength,
		uv.passwordRequired,
		uv.passwordLength,
		uv.passwordHash,
		uv.emailRequired,
		uv.normaliseEmail,
		uv.emailFormat,
		uv.emailIsTaken,
		uv.roleIDExists,
	); err != nil {
		return err
	}

	return uv.UserDB.Create(u)
}

func (uv *userValidator) Update(u *User) error {
	defer func() {
		u.Password = ""
	}()

	// we can then use the standard validation process here.
	uc := userValWithCurrent{uv: uv}
	if err := uv.runValFuncs(u,
		uc.fetchUser,
		uv.idNotAdmin,
		uv.firstNameRequired,
		uv.firstNameLength,
		uv.settingsLength,
		uv.emailRequired,
		uv.normaliseEmail,
		uv.emailFormat,
		uv.passwordLength,
		uv.passwordHash,
		uc.preservePassword,
		uc.emailIsTaken,
		uv.roleIDExists,
	); err != nil {
		return err
	}

	return uv.UserDB.Update(u)
}

func (uv *userValidator) Delete(id int64) error {
	if err := uv.runValFuncs(&User{ID: id},
		uv.idNotAdmin,
	); err != nil {
		return err
	}

	return uv.UserDB.Delete(id)
}

func (uv *userValidator) ByEmail(e string) (User, error) {
	user := User{
		Email: e,
	}

	if err := uv.runValFuncs(&user,
		uv.emailRequired,
		uv.normaliseEmail,
		uv.emailFormat,
	); err != nil {
		return User{}, err
	}

	return uv.UserDB.ByEmail(user.Email)
}

type userValFn func(u *User) error

type userValWithCurrent struct {
	uv      *userValidator
	current User
}

// fetchUser must be called before any of the other validators implemented by the receiver type. It
// retrieves the current user value from the database.
func (uc *userValWithCurrent) fetchUser() (string, userValFn) {
	return "", func(u *User) error {
		var err error
		uc.current, err = uc.uv.ByID(u.ID)
		if err != nil {
			return err
		}

		return nil
	}
}

// emailIsTaken makes sure u.Email is not taken in the database by other user that is not the one being updated
// now. It returns nil if the address is not taken. It may return ErrDuplicate.
func (uc *userValWithCurrent) emailIsTaken() (string, userValFn) {
	return "email", func(u *User) error {
		if uc.current.Email != u.Email {
			cu, err := uc.uv.UserDB.ByEmail(u.Email)
			if err == nil && u.ID != 0 && u.ID != cu.ID {
				return ErrDuplicate
			}
		}

		return nil
	}
}

// preservePassword makes sure an existing user's password is preserved if a new one is not provided.
// It does not return any errors.
//
// This method must be called AFTER the password hashing validators as it preserves the previous password for
// application users.
func (uc *userValWithCurrent) preservePassword() (string, userValFn) {
	return "", func(u *User) error {
		if u.Password == "" {
			u.Password = uc.current.Password
		}

		return nil
	}
}

func (uv *userValidator) runValFuncs(u *User, fns ...func() (string, userValFn)) error {
	return runValidationFunctions(u, fns)
}

// idSetToZero sets the user's ID to 0. It does not return any errors.
func (uv *userValidator) idSetToZero() (string, userValFn) {
	return "", func(u *User) error {
		u.ID = 0
		return nil
	}
}

// idNotAdmin makes sure a user's ID is not 1, the default admin user.
// It returns an ErrIDAdminCannotChange in case it is.
func (uv *userValidator) idNotAdmin() (string, userValFn) {
	return "", func(u *User) error {
		if u.ID == 1 {
			return ErrReadOnly
		}

		return nil
	}
}

// passwordRequired makes sure u.Password is not empty. It may return ErrRequired.
func (uv *userValidator) passwordRequired() (string, userValFn) {
	return "password", func(u *User) error {
		if u.Password == "" {
			return ErrRequired
		}

		return nil
	}
}

// passwordLength makes sure u.Password has at least 8 characters. It may return ErrTooShort
func (uv *userValidator) passwordLength() (string, userValFn) {
	return "password", func(u *User) error {
		if u.Password != "" && len(u.Password) < 8 {
			return ErrTooShort
		}

		return nil
	}
}

// passwordHash hashes the password. It may return private errors.
func (uv *userValidator) passwordHash() (string, userValFn) {
	return "", func(u *User) error {
		if u.Password == "" {
			return nil
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost+2)
		if err != nil {
			return wrap("failed to hash password", err)
		}

		u.Password = string(hash)

		return nil
	}
}

// normalizeEmail modifies u.Email to remove excess space and have all characters lowercase.
func (uv *userValidator) normaliseEmail() (string, userValFn) {
	return "email", func(u *User) error {
		u.Email = strings.ToLower(u.Email)
		u.Email = strings.TrimSpace(u.Email)
		return nil
	}
}

// emailRequired makes sure u.Email address is not empty. It may return ErrRequired.
func (uv *userValidator) emailRequired() (string, userValFn) {
	return "email", func(u *User) error {
		if u.Email == "" {
			return ErrRequired
		}

		return nil
	}
}

// emailFormat makes sure u.Email looks like an email address. It returns nil if the address
// is empty. It may return ErrInvalid.
func (uv *userValidator) emailFormat() (string, userValFn) {
	return "email", func(u *User) error {
		if u.Email == "" {
			return nil
		}

		if !uv.emailRegex.MatchString(u.Email) {
			return ErrInvalid
		}
		return nil
	}
}

// emailIsTaken makes sure u.Email is not taken in the database. It returns nil if the address
// is not taken. It may return ErrDuplicate.
func (uv *userValidator) emailIsTaken() (string, userValFn) {
	return "email", func(u *User) error {
		_, err := uv.UserDB.ByEmail(u.Email)
		if err == nil {
			return ErrDuplicate
		}

		return nil
	}
}

// firstNameRequired makes sure u.FirstName is not empty. It may return ErrRequired.
func (uv *userValidator) firstNameRequired() (string, userValFn) {
	return "firstName", func(u *User) error {
		if u.FirstName == "" {
			return ErrRequired
		}

		return nil
	}
}

// firstNameLength makes sure u.FirstName has at least two characters. It may return ErrTooShort.
func (uv *userValidator) firstNameLength() (string, userValFn) {
	return "firstName", func(u *User) error {
		if len(u.FirstName) < 2 {
			return ErrTooShort
		}

		return nil
	}
}

// settingsLength makes sure that the text contained in settings is not greater
// than X bytes. It may return ErrTooLong.
func (uv *userValidator) settingsLength() (string, userValFn) {
	return "settings", func(u *User) error {
		if len(u.Settings) > 8192 {
			return ErrTooLong
		}

		return nil
	}
}

// roleIDExists verifies that the requested role ID exists in the database. It may return
// an ErrNotFound.
//
// If an error happens when checking the database, it will be ignored and the validation will be considered
// as a pass. The database integrity checks will then make sure that the role exists in the end.
func (uv *userValidator) roleIDExists() (string, userValFn) {
	return "roleId", func(u *User) error {
		if _, err := uv.roleService.ByID(u.RoleID); err != nil {
			if xerrors.Is(err, ErrNotFound) {
				return ErrRefNotFound
			}
		}

		return nil
	}
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
