package models

import (
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/xerrors"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type testUserDB struct {
	UserDB
	byEmail func(e string) (User, error)
	byID    func(id int64) (User, error)
	byIDs   func(id ...int64) ([]User, error)
	delete  func(id int64) error
	create  func(*User) error
	update  func(*User) error
}

func (t *testUserDB) ByEmail(e string) (User, error) {
	if t.byEmail != nil {
		return t.byEmail(e)
	}

	return User{}, nil
}

func (t *testUserDB) ByID(id int64) (User, error) {
	if t.byID != nil {
		return t.byID(id)
	}

	return User{}, nil
}

func (t *testUserDB) ByIDs(id ...int64) ([]User, error) {
	if t.byIDs != nil {
		return t.byIDs(id...)
	}

	return nil, nil
}

func (t *testUserDB) Delete(id int64) error {
	if t.delete != nil {
		return t.delete(id)
	}

	return nil
}

func (t *testUserDB) Create(u *User) error {
	if t.create != nil {
		return t.create(u)
	}

	return nil
}

func (t *testUserDB) Update(u *User) error {
	if t.update != nil {
		return t.update(u)
	}

	return nil
}

func dropUsersTable(db *gorm.DB) {
	db.DropTableIfExists(&Rating{}, &User{})
}

type testSigner struct {
	jose.Signer
}

func (t *testSigner) Sign(b []byte) (*jose.JSONWebSignature, error) {
	return nil, privateError("this failed")
}

func TestNewUser(t *testing.T) {
	user := NewUser()
	assert.Equal(t, User{
		Active: true,
		RoleID: 2,
	}, user)
}

func TestUserService_Authenticate(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	var cases = []struct {
		name     string
		username string
		password string

		outerr error

		setup func()
	}{
		{
			"emptyUname",
			"",
			"password",
			ErrNoCredentials,
			nil,
		},
		{
			"emptyPass",
			"none@none.com",
			"",
			ErrNoCredentials,
			nil,
		},
		{
			"badUname",
			"hslkdjfghlskjdf",
			"password",
			ErrUnauthorised,
			nil,
		},
		{
			"badPass",
			"noone@someone.com",
			"pass",
			ErrUnauthorised,
			nil,
		},
		{
			"noUser",
			"  AUSEREMAIL@name.COM  ",
			"password",
			ErrUnauthorised,
			func() {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "auseremail@name.com", e)

					return User{}, ErrNotFound
				}
			},
		},
		{
			"dbInternalError",
			"  AUSEREMAIL@name.COM  ",
			"password",
			privateError("test private error"),
			func() {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "auseremail@name.com", e)

					return User{}, privateError("test private error")
				}
			},
		},
		{
			"passwordNotMatch",
			"  AUSEREMAIL@name.COM  ",
			"password",
			ErrUnauthorised,
			func() {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "auseremail@name.com", e)

					hash, err := bcrypt.GenerateFromPassword([]byte("adifferentpassword"), bcrypt.DefaultCost+2)
					if err != nil {
						return User{}, wrap("failed to hash password", err)
					}

					return User{
						Active:   true,
						Password: string(hash),
					}, nil
				}
			},
		},
		{
			"userInactive",
			"auseremail@name.com",
			"7vb6sCaHrV5DfV6wE7i9QdGC",
			ErrUnauthorised,
			func() {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "auseremail@name.com", e)

					hash, err := bcrypt.GenerateFromPassword([]byte("7vb6sCaHrV5DfV6wE7i9QdGC"), bcrypt.DefaultCost+2)
					if err != nil {
						return User{}, wrap("failed to hash password", err)
					}

					return User{
						ID:       99,
						Active:   false,
						Password: string(hash),
					}, nil
				}
			},
		},
		{
			"passwordMatch",
			"  AUSEREMAIL@name.COM  ",
			"7vb6sCaHrV5DfV6wE7i9QdGC",
			nil,
			func() {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "auseremail@name.com", e)

					hash, err := bcrypt.GenerateFromPassword([]byte("7vb6sCaHrV5DfV6wE7i9QdGC"), bcrypt.DefaultCost+2)
					if err != nil {
						return User{}, wrap("failed to hash password", err)
					}

					return User{
						ID:       99,
						Active:   true,
						Password: string(hash),
					}, nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			if cs.setup != nil {
				cs.setup()
			}

			user, err := us.Authenticate(cs.username, cs.password)

			if cs.outerr != nil {
				assert.Error(t, err)
				assert.True(t, xerrors.Is(err, cs.outerr), "errors must match, expected %v, got %v", cs.outerr, err)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(99), user.ID)
			}

			tudb.byEmail = nil
		})
	}
}

func TestUserService_Refresh(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	t.Run("noToken", func(t *testing.T) {
		_, err := us.Refresh("")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNoCredentials))
	})

	t.Run("badToken", func(t *testing.T) {
		_, err := us.Refresh("very.bad.token")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("expiredToken", func(t *testing.T) {
		clr := authClaims{
			Claims: jwt.Claims{
				Subject: "999",
				Issuer:  "ratingsappr",
				Expiry:  jwt.NewNumericDate(time.Now().UTC().Add(-10 * time.Minute)),
			},
			RoleID: 888,
		}

		rtok, err := jwt.Signed(us.(*userService).signer).Claims(clr).CompactSerialize()
		require.NoError(t, err)

		_, err = us.Refresh(rtok)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("wrongTokenType", func(t *testing.T) {
		tok, err := us.Token(&User{
			ID:     888,
			Active: true,
			RoleID: 999,
		})
		require.NoError(t, err)

		_, err = us.Refresh(tok.AccessToken)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("notFound", func(t *testing.T) {
		tok, err := us.Token(&User{
			ID:     888,
			Active: true,
			RoleID: 999,
		})
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return User{}, ErrNotFound
		}

		_, err = us.Refresh(tok.RefreshToken)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("dbErrorInternal", func(t *testing.T) {
		tok, err := us.Token(&User{
			ID:     888,
			Active: true,
			RoleID: 999,
		})
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return User{}, wrap("some internal error", nil)
		}

		_, err = us.Refresh(tok.RefreshToken)

		assert.Error(t, err)
	})

	t.Run("inactiveUser", func(t *testing.T) {
		user := User{
			ID:     888,
			Active: true,
			RoleID: 999,
		}

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)

			ret := user
			ret.Active = false
			return ret, nil
		}

		tok, err := us.Token(&user)
		require.NoError(t, err)

		_, err = us.Refresh(tok.RefreshToken)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("ok", func(t *testing.T) {
		user := User{
			ID:     888,
			Active: true,
			RoleID: 999,
		}

		tok, err := us.Token(&user)
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return user, nil
		}

		ruser, err := us.Refresh(tok.RefreshToken)

		assert.NoError(t, err)
		assert.Equal(t, user, ruser)
	})
}

func TestUserService_Validate(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	t.Run("noToken", func(t *testing.T) {
		_, err := us.Validate("")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("badToken", func(t *testing.T) {
		_, err := us.Validate("very.bad.token")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("expiredToken", func(t *testing.T) {
		clr := authClaims{
			Claims: jwt.Claims{
				Subject: "999",
				Issuer:  "ratingsapp",
				Expiry:  jwt.NewNumericDate(time.Now().UTC().Add(-10 * time.Minute)),
			},
			RoleID: 888,
		}

		atok, err := jwt.Signed(us.(*userService).signer).Claims(clr).CompactSerialize()
		require.NoError(t, err)

		_, err = us.Validate(atok)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("wrongTokenType", func(t *testing.T) {
		tok, err := us.Token(&User{
			ID:     888,
			Active: true,
			RoleID: 999,
		})
		require.NoError(t, err)

		_, err = us.Validate(tok.RefreshToken)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("notFound", func(t *testing.T) {
		tok, err := us.Token(&User{
			ID:     888,
			Active: true,
			RoleID: 999,
		})
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return User{}, ErrNotFound
		}

		_, err = us.Validate(tok.AccessToken)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("dbErrorInternal", func(t *testing.T) {
		tok, err := us.Token(&User{
			ID:     888,
			Active: true,
			RoleID: 999,
		})
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return User{}, wrap("some error message", nil)
		}

		_, err = us.Validate(tok.AccessToken)

		assert.Error(t, err)
	})

	t.Run("inactiveUser", func(t *testing.T) {
		user := User{
			ID:     888,
			Active: true,
			RoleID: 999,
		}

		tok, err := us.Token(&user)
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)

			ret := user
			ret.Active = false
			return ret, nil
		}

		_, err = us.Validate(tok.AccessToken)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrUnauthorised))
	})

	t.Run("ok", func(t *testing.T) {
		user := User{
			ID:     888,
			Active: true,
			RoleID: 999,
		}

		tok, err := us.Token(&user)
		require.NoError(t, err)

		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return user, nil
		}

		auser, err := us.Validate(tok.AccessToken)

		assert.NoError(t, err)
		assert.Equal(t, user, auser)
	})
}

func TestUserService_Token(t *testing.T) {
	const jwtkey = "test secret key for jwt signing"

	us, _ := NewUserService(nil, nil, []byte(jwtkey))
	user := User{
		ID:     999,
		RoleID: 888,
	}

	assert.True(t, jwtAccessDuration >= 6*time.Hour, "jwt access duration must have a reasonable length of time")
	assert.True(t, jwtRefreshDuration >= 1*24*time.Hour, "jwt refresh duration must have a reasonable length of time")

	t.Run("good", func(t *testing.T) {
		tok, err := us.Token(&user)
		assert.NoError(t, err)
		assert.NotEmpty(t, tok.AccessToken)
		assert.NotEmpty(t, tok.RefreshToken)
		assert.Equal(t, "bearer", tok.TokenType)
		assert.Equal(t, int(jwtAccessDuration/time.Second), tok.ExpiresIn)

		// access token
		jtok, err := jwt.ParseSigned(tok.AccessToken)
		require.NoError(t, err, "token must be well formed")

		var cl = authClaims{}
		require.NoError(t, jtok.Claims([]byte(jwtkey), &cl), "token must be parseable")
		assert.NoError(t, cl.Validate(jwt.Expected{
			Issuer: "ratingsapp",
			Time:   time.Now().UTC(),
		}), "token is valid against key")

		assert.Equal(t, "999", cl.Subject, "subject is present on token")
		assert.Equal(t, int64(888), cl.RoleID, "role id is included in token")
		assert.True(t, cl.Expiry.Time().After(time.Now().Add(jwtAccessDuration-1*time.Minute)), "token has the right expiry time")
		assert.True(t, cl.Expiry.Time().Before(time.Now().Add(jwtAccessDuration+1*time.Minute)), "token has the right expiry time")

		rtok, err := jwt.ParseSigned(tok.RefreshToken)
		require.NoError(t, err)

		cl = authClaims{}
		require.NoError(t, rtok.Claims([]byte(jwtkey), &cl), "refresh token must be parseable")
		assert.NoError(t, cl.Validate(jwt.Expected{
			Issuer: "ratingsappr",
			Time:   time.Now().UTC(),
		}), "refresh token is valid against key")

		require.Equal(t, "999", cl.Subject, "subject is present on token")
		require.Equal(t, int64(888), cl.RoleID, "role id is included in token")
		assert.True(t, cl.Expiry.Time().After(time.Now().Add(jwtRefreshDuration-1*time.Minute)), "token has the right expiry time")
		assert.True(t, cl.Expiry.Time().Before(time.Now().Add(jwtRefreshDuration+1*time.Minute)), "token has the right expiry time")
	})

	t.Run("badSigner", func(t *testing.T) {
		us.(*userService).signer = &testSigner{}

		tok, err := us.Token(&user)
		assert.Error(t, err)
		assert.Equal(t, Token{}, tok)
	})

}

func TestUserService_ByID(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	t.Run("hidePassword", func(t *testing.T) {
		user := User{
			ID:       888,
			Active:   true,
			Password: "somesupersecrethashofthepassword",
		}
		tudb.byID = func(id int64) (User, error) {
			assert.Equal(t, int64(888), id)
			return user, nil
		}

		ruser, err := us.ByID(888)
		user.Password = ""

		assert.NoError(t, err)
		assert.Equal(t, user, ruser)
	})

}

func TestUserService_ByEmail(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	t.Run("hidePassword", func(t *testing.T) {
		user := User{
			ID:       888,
			Active:   true,
			Email:    "test@example.com",
			Password: "somesupersecrethashofthepassword",
		}
		tudb.byEmail = func(e string) (User, error) {
			assert.Equal(t, "test@example.com", e)
			return user, nil
		}

		ruser, err := us.ByEmail("test@example.com")
		user.Password = ""

		assert.NoError(t, err)
		assert.Equal(t, user, ruser)
	})

	t.Run("badEmailAddress", func(t *testing.T) {
		tudb.byEmail = func(e string) (User, error) {
			assert.Equal(t, "thldfkghhsdfkljmple.com", e)
			return User{}, nil
		}

		_, err := us.ByEmail("thldfkghhsdfkljmple.com")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"email": ErrInvalid}))
	})

	t.Run("emptyEmailAddress", func(t *testing.T) {
		tudb.byEmail = func(e string) (User, error) {
			assert.Equal(t, "", e)
			return User{}, nil
		}

		_, err := us.ByEmail("")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"email": ErrRequired}))
	})

}

func TestUserService_ByIDs(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	t.Run("hidePasswords", func(t *testing.T) {
		users := []User{
			User{
				ID:       888,
				Active:   true,
				Password: "somesupersecrethashofthepassword",
			},
			User{
				ID:       999,
				Active:   true,
				Password: "somesupersecrethashoftheotherpassword",
			},
		}
		tudb.byIDs = func(id ...int64) ([]User, error) {
			var ret [2]User
			assert.Equal(t, int64(888), id[0])
			assert.Equal(t, int64(999), id[1])

			copy(ret[:], users)
			return ret[:], nil
		}

		rusers, err := us.ByIDs(888, 999)
		users[0].Password = ""
		users[1].Password = ""

		assert.NoError(t, err)
		assert.Equal(t, users, rusers)
	})
}

func TestUserService_Delete(t *testing.T) {
	tudb := &testUserDB{}
	us, _ := NewUserService(nil, nil, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	t.Run("mustNotDeleteAdmin", func(t *testing.T) {
		tudb.delete = nil

		err := us.Delete(1)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrReadOnly))
	})

	t.Run("ok", func(t *testing.T) {
		var called bool
		tudb.delete = func(id int64) error {
			assert.Equal(t, int64(888), id)
			called = true
			return nil
		}

		err := us.Delete(888)

		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestUserService_Create(t *testing.T) {
	rdb := &testRoleDB{}
	rs := NewRoleService(nil)
	rs.(*roleService).RoleService.(*roleValidator).RoleDB = rdb

	tudb := &testUserDB{}
	us, _ := NewUserService(nil, rs, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	goodEmail := func(e string) (User, error) {
		return User{}, ErrNotFound
	}
	var created bool
	goodCreate := func(u *User) error {
		assert.NotEmpty(t, u.Password)
		created = true
		return nil
	}
	tudb.byEmail = goodEmail
	tudb.create = goodCreate

	var cases = []struct {
		name    string
		user    *User
		outuser *User
		outerr  error
		setup   func(*testing.T)
	}{
		{
			"idMustBeZero",
			&User{ID: 99, Email: "test@address.com", FirstName: "Test", Password: "testpassword"},
			&User{ID: 0, Email: "test@address.com", FirstName: "Test", Password: ""},
			nil,
			nil,
		},
		{
			"emailRequired",
			&User{RoleID: 2, Email: "", FirstName: "Test", Password: "testpassword"},
			nil,
			ValidationError{"email": ErrRequired},
			nil,
		},
		{
			"emailNoMatch",
			&User{RoleID: 2, Email: "  testEmailBad!!##BADEMAIL   ", FirstName: "Test", Password: "testpassword"},
			nil,
			ValidationError{"email": ErrInvalid},
			nil,
		},
		{
			"emailTaken",
			&User{RoleID: 2, Email: "TEST@ADDRESS.COM", FirstName: "Test", Password: "testpassword"},
			nil,
			ValidationError{"email": ErrDuplicate},
			func(t *testing.T) {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "test@address.com", e)
					return User{}, nil
				}
			},
		},
		{
			"emailTakenFails",
			&User{RoleID: 2, Email: "TEST@ADDRESS.COM", FirstName: "Test", Password: "testpassword"},
			&User{RoleID: 2, Email: "test@address.com", FirstName: "Test", Password: ""},
			nil,
			func(t *testing.T) {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "test@address.com", e)
					return User{}, wrap("test error", nil)
				}
			},
		},
		{
			"emailNormalizes",
			&User{RoleID: 2, Email: "    A_TEST@ADDRESS.COM   ", FirstName: "Test", Password: "testpassword"},
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "Test", Password: ""},
			nil,
			nil,
		},
		{
			"firstNameRequired",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "", Password: "testpassword"},
			nil,
			ValidationError{"firstName": ErrRequired},
			nil,
		},
		{
			"firstNameLength",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "s", Password: "testpassword"},
			nil,
			ValidationError{"firstName": ErrTooShort},
			nil,
		},
		{
			"settingsLength",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "shortname",
				Password: "testpassword", Settings: `{"setting1": "` + strings.Repeat("a", 8192) + `"}`},
			nil,
			ValidationError{"settings": ErrTooLong},
			nil,
		},
		{
			"passwordRequired",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "shortname", Password: ""},
			nil,
			ValidationError{"password": ErrRequired},
			nil,
		},
		{
			"passwordLength",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "shortname", Password: "assword"},
			nil,
			ValidationError{"password": ErrTooShort},
			nil,
		},
		{
			"roleIDNotFound",
			&User{RoleID: 0, Email: "a_test@address.com", FirstName: "shortname", Password: "lksdjhdfkgljhslkdjhfg"},
			nil,
			ValidationError{"roleId": ErrRefNotFound},
			func(t *testing.T) {
				rdb.byID = func(id int64) (Role, error) {
					assert.Equal(t, int64(0), id)
					return Role{}, ErrNotFound
				}
			},
		},
		{
			"multipleErrors",
			&User{RoleID: 2, Email: "a_teksjhdflgkj", FirstName: "", Password: "gf"},
			nil,
			ValidationError{"email": ErrInvalid, "firstName": ErrRequired, "password": ErrTooShort},
			nil,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			if cs.setup != nil {
				cs.setup(t)
			}

			err := us.Create(cs.user)

			if cs.outerr != nil {
				assert.Error(t, err)
				assert.True(t, xerrors.Is(err, cs.outerr), "errors must match, expected %v, got %v", cs.outerr, err)

			} else {
				assert.Equal(t, cs.outuser, cs.user)
				assert.True(t, created)
			}

			*tudb = testUserDB{}
			*rdb = testRoleDB{}
			tudb.create, tudb.byEmail = goodCreate, goodEmail
			created = false
		})
	}
}

func TestUserService_Update(t *testing.T) {
	rdb := &testRoleDB{}
	rs := NewRoleService(nil)
	rs.(*roleService).RoleService.(*roleValidator).RoleDB = rdb

	tudb := &testUserDB{}
	us, _ := NewUserService(nil, rs, []byte(testJWTSecret))
	us.(*userService).UserService.(*userValidator).UserDB = tudb

	goodEmail := func(e string) (User, error) {
		return User{}, ErrNotFound
	}
	var updated bool
	goodUpdate := func(u *User) error {
		assert.NotEmpty(t, u.Password)
		updated = true
		return nil
	}
	tudb.byEmail = goodEmail
	tudb.update = goodUpdate

	var cases = []struct {
		name    string
		user    *User
		outuser *User
		outerr  error
		setup   func(*testing.T)
	}{
		{
			"cannotFind",
			&User{ID: 99, Email: "test@address.com", FirstName: "Test", Password: "testpassword"},
			nil,
			ErrNotFound,
			func(t *testing.T) {
				tudb.byID = func(id int64) (User, error) {
					assert.Equal(t, int64(99), id)
					return User{}, ErrNotFound
				}
			},
		},
		{
			"idIsAdmin",
			&User{ID: 1, Email: "test@address.com", FirstName: "Test", Password: "testpassword"},
			nil,
			ErrReadOnly,
			nil,
		},
		{
			"emailRequired",
			&User{RoleID: 2, Email: "", FirstName: "Test", Password: "testpassword"},
			nil,
			ValidationError{"email": ErrRequired},
			nil,
		},
		{
			"emailNoMatch",
			&User{RoleID: 2, Email: "  testEmailBad!!##BADEMAIL   ", FirstName: "Test", Password: "testpassword"},
			nil,
			ValidationError{"email": ErrInvalid},
			nil,
		},
		{
			"emailTaken",
			&User{ID: 10, RoleID: 2, Email: "TEST@ADDRESS.COM", FirstName: "Test", Password: "testpassword"},
			nil,
			ValidationError{"email": ErrDuplicate},
			func(t *testing.T) {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "test@address.com", e)
					return User{ID: 13}, nil
				}
			},
		},
		{
			"emailNotTaken",
			&User{ID: 10, RoleID: 2, Email: "TEST@ADDRESS.COM", FirstName: "Test", Password: "testpassword"},
			&User{ID: 10, RoleID: 2, Email: "test@address.com", FirstName: "Test", Password: ""},
			nil,
			func(t *testing.T) {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "test@address.com", e)
					return User{ID: 10}, nil
				}
			},
		},
		{
			"emailTakenFails",
			&User{RoleID: 2, Email: "TEST@ADDRESS.COM", FirstName: "Test", Password: "testpassword"},
			&User{RoleID: 2, Email: "test@address.com", FirstName: "Test", Password: ""},
			nil,
			func(t *testing.T) {
				tudb.byEmail = func(e string) (User, error) {
					assert.Equal(t, "test@address.com", e)
					return User{}, wrap("test error", nil)
				}
			},
		},
		{
			"emailNormalizes",
			&User{RoleID: 2, Email: "    A_TEST@ADDRESS.COM   ", FirstName: "Test", Password: "testpassword"},
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "Test", Password: ""},
			nil,
			nil,
		},
		{
			"firstNameRequired",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "", Password: "testpassword"},
			nil,
			ValidationError{"firstName": ErrRequired},
			nil,
		},
		{
			"firstNameLength",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "s", Password: "testpassword"},
			nil,
			ValidationError{"firstName": ErrTooShort},
			nil,
		},
		{
			"settingsLength",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "shortname",
				Password: "testpassword", Settings: `{"setting1": "` + strings.Repeat("a", 8192) + `"}`},
			nil,
			ValidationError{"settings": ErrTooLong},
			nil,
		},
		{
			"passwordNoChange",
			&User{ID: 99, Email: "test@address.com", FirstName: "AnotherTest", Password: ""},
			&User{ID: 99, Email: "test@address.com", FirstName: "AnotherTest", Password: ""},
			nil,
			func(t *testing.T) {
				tudb.byID = func(id int64) (User, error) {
					assert.Equal(t, int64(99), id)
					return User{
						ID:        99,
						Email:     "test@address.com",
						FirstName: "Test",
						Password:  "passwordHash",
					}, nil
				}
				tudb.update = func(u *User) error {
					updated = true

					assert.Equal(t, &User{
						ID:        99,
						Email:     "test@address.com",
						FirstName: "AnotherTest",
						Password:  "passwordHash",
					}, u)

					return nil
				}
			},
		},
		{
			"passwordLength",
			&User{RoleID: 2, Email: "a_test@address.com", FirstName: "shortname", Password: "assword"},
			nil,
			ValidationError{"password": ErrTooShort},
			nil,
		},
		{
			"changedPasswordIsHashed",
			&User{ID: 99, Email: "test@address.com", FirstName: "AnotherTest", Password: "newPassword"},
			&User{ID: 99, Email: "test@address.com", FirstName: "AnotherTest", Password: ""},
			nil,
			func(t *testing.T) {
				tudb.byID = func(id int64) (User, error) {
					assert.Equal(t, int64(99), id)
					return User{ID: 99,
						Email:     "test@address.com",
						FirstName: "Test",
						Password:  "aPasswordHash",
					}, nil
				}
				tudb.update = func(u *User) error {
					updated = true

					assert.NotEmpty(t, u.Password, "password is hashed")
					assert.NotEqual(t, "aPasswordHash", u.Password, "password is hashed")
					assert.NotEqual(t, "newPassword", u.Password, "password is hashed")
					assert.Equal(t, &User{
						ID:        99,
						Email:     "test@address.com",
						FirstName: "AnotherTest",
						Password:  u.Password,
					}, u)

					return nil
				}
			},
		},
		{
			"roleIDNotFound",
			&User{RoleID: 0, Email: "a_test@address.com", FirstName: "shortname", Password: "lksdjhdfkgljhslkdjhfg"},
			nil,
			ValidationError{"roleId": ErrRefNotFound},
			func(t *testing.T) {
				rdb.byID = func(id int64) (Role, error) {
					assert.Equal(t, int64(0), id)
					return Role{}, ErrNotFound
				}
			},
		},
		{
			"multipleErrors",
			&User{RoleID: 2, Email: "a_teksjhdflgkj", FirstName: "", Password: "gf"},
			nil,
			ValidationError{"email": ErrInvalid, "firstName": ErrRequired, "password": ErrTooShort},
			nil,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			if cs.setup != nil {
				cs.setup(t)
			}

			err := us.Update(cs.user)

			if cs.outerr != nil {
				assert.Error(t, err)
				assert.True(t, xerrors.Is(err, cs.outerr), "errors must match, expected %v, got %v", cs.outerr, err)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, cs.outuser, cs.user)
				assert.True(t, updated)
			}

			*tudb = testUserDB{}
			*rdb = testRoleDB{}
			tudb.update, tudb.byEmail = goodUpdate, goodEmail
			updated = false
		})
	}
}

func TestUserGORM_Create(t *testing.T) {
	t.Run("idExists", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)

		err := (&userGorm{db}).Create(user)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"id": ErrIDTaken}))
	})

	t.Run("emailTaken", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)

		user.ID = 0
		err := (&userGorm{db}).Create(user)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"email": ErrDuplicate}))
	})

	t.Run("roleIDInvalid", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			Active:    true,
			RoleID:    88,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		err := (&userGorm{db}).Create(user)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"roleId": ErrRefNotFound}))
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			Active:    true,
			RoleID:    1,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		err := (&userGorm{db}).Create(user)

		assert.NoError(t, err)
		assert.NotEqual(t, 0, user.ID)

		var count int
		db.Model(&User{}).Where("id = ?", user.ID).Count(&count)
		assert.Equal(t, 1, count)
	})
}

func TestUserGORM_Update(t *testing.T) {
	t.Run("idNotExists", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		err := (&userGorm{db}).Update(user)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("noChanges", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)

		err := (&userGorm{db}).Update(user)
		assert.NoError(t, err)

		var cuser User
		require.NoError(t, db.First(&cuser, 10).Error)
		assert.Equal(t, user, &cuser)

	})

	t.Run("changesDefaultGoValues", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)
		user.Active = false
		user.FirstName = ""
		user.LastName = ""
		user.Password = ""
		user.Settings = ""

		err := (&userGorm{db}).Update(user)
		assert.NoError(t, err)

		var cuser User
		require.NoError(t, db.First(&cuser, 10).Error)
		assert.Equal(t, user, &cuser)

	})

	t.Run("emailTaken", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)
		user.ID = 11
		user.Email = "different@email.com"
		require.NoError(t, db.Create(user).Error)

		user.Email = "test@test.com"
		err := (&userGorm{db}).Update(user)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"email": ErrDuplicate}))
	})

	t.Run("roleIDInvalid", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)

		user.RoleID = 88
		err := (&userGorm{db}).Update(user)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ValidationError{"roleId": ErrRefNotFound}))
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        10,
			Active:    true,
			RoleID:    2,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}

		require.NoError(t, db.Create(user).Error)

		user.Active = false
		user.RoleID = 1
		user.Email = "another@test.com"
		user.FirstName = "Another"
		user.LastName = ""
		user.Password = "Different Hash"
		user.Settings = "Changed settings"

		err := (&userGorm{db}).Update(user)

		assert.NoError(t, err)

		var cuser User
		require.NoError(t, db.First(&cuser, 10).Error)
		assert.Equal(t, user, &cuser)
	})
}

func TestUserGORM_Delete(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		err := (&userGorm{db}).Delete(999)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropUsersTable(db)

		err := (&userGorm{db}).Delete(999)

		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        999,
			RoleID:    99,
			Active:    true,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}
		role := &Role{
			ID:          99,
			Label:       "test",
			Permissions: 7,
		}

		require.NoError(t, db.Create(role).Error)
		require.NoError(t, db.Create(user).Error)

		err := (&userGorm{db}).Delete(999)
		user.Role = role

		assert.NoError(t, err)
	})
}

func TestUserGORM_ByEmail(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		_, err := (&userGorm{db}).ByEmail("atestaddress")

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropUsersTable(db)

		_, err := (&userGorm{db}).ByEmail("atestaddress")

		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			RoleID:    99,
			Active:    true,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}
		role := &Role{
			ID:          99,
			Label:       "test",
			Permissions: 15,
		}

		require.NoError(t, db.Create(role).Error)
		require.NoError(t, db.Create(user).Error)

		outuser, err := (&userGorm{db}).ByEmail("test@test.com")
		user.Role = role

		assert.NoError(t, err)
		assert.Equal(t, user, &outuser)
		assert.Equal(t, role, outuser.Role, "must preload the role")
	})
}

func TestUserGORM_ByID(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		_, err := (&userGorm{db}).ByID(999)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropUsersTable(db)

		_, err := (&userGorm{db}).ByID(999)

		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:        999,
			RoleID:    99,
			Active:    true,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}
		role := &Role{
			ID:          99,
			Label:       "test",
			Permissions: 7,
		}

		require.NoError(t, db.Create(role).Error)
		require.NoError(t, db.Create(user).Error)

		outuser, err := (&userGorm{db}).ByID(999)
		user.Role = role

		assert.NoError(t, err)
		assert.Equal(t, user, &outuser)
		assert.Equal(t, role, outuser.Role, "must preload the role")
	})
}

func TestUserGORM_ByIDs(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		users, err := (&userGorm{db}).ByIDs(999)

		assert.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropUsersTable(db)

		_, err := (&userGorm{db}).ByIDs(999)

		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		user1 := User{
			ID:        999,
			RoleID:    99,
			Active:    true,
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "TestPasswordHAsh",
			Settings:  "Settings string here",
		}
		user2 := User{
			ID:        1002,
			RoleID:    99,
			Active:    true,
			Email:     "second@test.com",
			FirstName: "Second",
			LastName:  "User",
			Password:  "TestPasswordHAshOther",
			Settings:  "Settings string here",
		}
		role := Role{
			ID:          99,
			Label:       "test",
			Permissions: 7,
		}

		require.NoError(t, db.Create(&role).Error)
		require.NoError(t, db.Create(&user1).Error)
		require.NoError(t, db.Create(&user2).Error)

		t.Run("listAll", func(t *testing.T) {
			outusers, err := (&userGorm{db}).ByIDs()

			assert.NoError(t, err)
			assert.Len(t, outusers, 3)
			assert.Contains(t, outusers, user1)
			assert.Contains(t, outusers, user2)
		})

		t.Run("listOne", func(t *testing.T) {
			outusers, err := (&userGorm{db}).ByIDs(999)

			assert.NoError(t, err)
			assert.Len(t, outusers, 1)
			assert.Contains(t, outusers, user1)
		})

		t.Run("listOther", func(t *testing.T) {
			outusers, err := (&userGorm{db}).ByIDs(1002)

			assert.NoError(t, err)
			assert.Len(t, outusers, 1)
			assert.Contains(t, outusers, user2)
		})

		t.Run("listSome", func(t *testing.T) {
			outusers, err := (&userGorm{db}).ByIDs(1002, 999)

			assert.NoError(t, err)
			assert.Len(t, outusers, 2)
			assert.Contains(t, outusers, user1)
			assert.Contains(t, outusers, user2)
		})
	})
}
