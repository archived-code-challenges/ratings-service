package models

import (
	"encoding/json"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

type testRoleDB struct {
	RoleDB
	create func(*Role) error
	update func(*Role) error
	delete func(int64) error
	byID   func(int64) (Role, error)
	byIDs  func(...int64) ([]Role, error)
}

func (t *testRoleDB) Create(mr *Role) error {
	if t.create != nil {
		return t.create(mr)
	}

	return nil
}

func (t *testRoleDB) Update(mr *Role) error {
	if t.update != nil {
		return t.update(mr)
	}

	return nil
}

func (t *testRoleDB) Delete(id int64) error {
	if t.delete != nil {
		return t.delete(id)
	}

	return nil
}

func (t *testRoleDB) ByID(id int64) (Role, error) {
	if t.byID != nil {
		return t.byID(id)
	}

	return Role{}, nil
}

func (t *testRoleDB) ByIDs(id ...int64) ([]Role, error) {
	if t.byIDs != nil {
		return t.byIDs(id...)
	}

	return []Role{}, nil
}

func dropRolesTable(db *gorm.DB) {
	db.DropTableIfExists(&Rating{}, &User{}, &Role{})
}

func TestPermissions_UnmarshalJSON(t *testing.T) {
	var cases = []struct {
		name   string
		in     []byte
		out    Permissions
		outerr bool
	}{
		{
			"invalidJSON1",
			[]byte(`][`),
			Permissions(0),
			true,
		},
		{
			"invalidJSON2",
			[]byte(`{}`),
			Permissions(0),
			true,
		},
		{
			"invalidJSON3",
			[]byte(``),
			Permissions(0),
			true,
		},
		{
			"empty",
			[]byte(`[]`),
			Permissions(0),
			false,
		},
		{
			"permNotExists",
			[]byte(`["readUsers","writeUsers","sillyStuff"]`),
			Permissions(0),
			true,
		},
		{
			"allPerms",
			[]byte(`["readUsers","writeUsers","readRatings","writeRatings"]`),
			PermissionReadUsers | PermissionWriteUsers | PermissionReadRatings | PermissionWriteRatings,
			false,
		},
		{
			"somePerms",
			[]byte(`["readUsers","readRatings"]`),
			PermissionReadUsers | PermissionReadRatings,
			false,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			var p Permissions

			err := json.Unmarshal(cs.in, &p)

			if cs.outerr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, cs.out, p)
			}
		})
	}
}

func TestPermissions_MarshalJSON(t *testing.T) {
	var cases = []struct {
		name string
		in   Permissions
		out  []byte
	}{
		{
			"empty",
			Permissions(0),
			[]byte(`[]`),
		},
		{
			"allPerms",
			PermissionReadUsers | PermissionWriteUsers | PermissionReadRatings | PermissionWriteRatings,
			[]byte(`["readUsers","writeUsers","readRatings","writeRatings"]`),
		},
		{
			"somePerms",
			PermissionReadUsers | PermissionReadRatings,
			[]byte(`["readUsers","readRatings"]`),
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			b, err := json.Marshal(cs.in)

			assert.NoError(t, err)
			assert.JSONEq(t, string(cs.out), string(b))
		})
	}
}

func TestRoleService_Create(t *testing.T) {
	rdb := &testRoleDB{}
	rs := NewRoleService(nil)
	rs.(*roleService).RoleService.(*roleValidator).RoleDB = rdb

	var cases = []struct {
		name    string
		role    *Role
		outrole *Role
		outerr  error
		setup   func(t *testing.T)
	}{
		{
			"labelAdminNotAllowed",
			&Role{Label: "admin", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrDuplicate},
			nil,
		},
		{
			"labelUserNotAllowed",
			&Role{Label: "user", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrDuplicate},
			nil,
		},
		{
			"labelEmpty",
			&Role{Label: "", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrRequired},
			nil,
		},
		{
			"labelTaken",
			&Role{Label: "new label", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrDuplicate},
			func(t *testing.T) {
				rdb.create = func(r *Role) error {
					cr := NewRole()
					cr.Label = "new label"
					cr.Permissions = PermissionWriteRatings

					assert.Equal(t, &cr, r)
					return ValidationError{"label": ErrDuplicate}
				}
			},
		},
		{
			"labelTooShort",
			&Role{Label: "dfg", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrTooShort},
			nil,
		},
		{
			"normalizes",
			&Role{Label: "    DFGSDFLKJlkjlk TEST   ", Permissions: PermissionWriteRatings},
			&Role{Label: "dfgsdflkjlkjlk test", Permissions: PermissionWriteRatings},
			nil,
			func(t *testing.T) {
				rdb.create = func(r *Role) error {
					cr := NewRole()
					cr.Label = "dfgsdflkjlkjlk test"
					cr.Permissions = PermissionWriteRatings

					assert.Equal(t, &cr, r)
					return nil
				}
			},
		},
		{
			"ok",
			&Role{Label: "alabel", Permissions: PermissionWriteRatings | PermissionReadRatings},
			&Role{ID: 99, Label: "alabel", Permissions: PermissionWriteRatings | PermissionReadRatings},
			nil,
			func(t *testing.T) {
				rdb.create = func(r *Role) error {
					cr := NewRole()
					cr.Label = "alabel"
					cr.Permissions = PermissionWriteRatings | PermissionReadRatings

					assert.Equal(t, &cr, r)
					r.ID = 99
					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			if cs.setup != nil {
				cs.setup(t)
			}

			err := rs.Create(cs.role)

			if cs.outerr != nil {
				assert.Error(t, err)
				assert.True(t, xerrors.Is(err, cs.outerr),
					"errors must match, expected %v, got %v", cs.outerr, err)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, cs.outrole, cs.role)
			}
		})
	}
}

func TestRoleService_Update(t *testing.T) {
	// Setup mock database
	rdb := &testRoleDB{}
	rs := NewRoleService(nil)
	rs.(*roleService).RoleService.(*roleValidator).RoleDB = rdb // Mock

	// Test Cases
	var cases = []struct {
		name    string
		role    *Role
		outrole *Role
		outerr  error
		setup   func(t *testing.T)
	}{
		{
			"labelAdminNotAllowed",
			&Role{ID: 99, Label: "admin", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrDuplicate},
			nil,
		},
		{
			"labelUserNotAllowed",
			&Role{ID: 99, Label: "user", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrDuplicate},
			nil,
		},
		{
			"adminIDReadOnly",
			&Role{ID: 1, Label: "update admin label", Permissions: PermissionReadUsers},
			nil,
			ErrReadOnly,
			nil,
		},
		{
			"userIDReadOnly",
			&Role{ID: 2, Label: "update admin label", Permissions: PermissionReadUsers},
			nil,
			ErrReadOnly,
			nil,
		},
		{
			"labelEmpty",
			&Role{ID: 99, Label: "", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrRequired},
			nil,
		},
		{
			"labelTaken",
			&Role{ID: 99, Label: "new label", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrDuplicate},
			func(t *testing.T) {
				rdb.update = func(r *Role) error {
					cr := NewRole()
					cr.ID = 99
					cr.Label = "new label"
					cr.Permissions = PermissionWriteRatings

					assert.Equal(t, &cr, r)
					return ValidationError{"label": ErrDuplicate}
				}
			},
		},
		{
			"labelTooShort",
			&Role{ID: 99, Label: "dfg", Permissions: PermissionWriteRatings},
			nil,
			ValidationError{"label": ErrTooShort},
			nil,
		},
		{
			"normalizes",
			&Role{ID: 99, Label: "    DFGSDFLKJlkjlk TEST   ", Permissions: PermissionWriteRatings},
			&Role{ID: 99, Label: "dfgsdflkjlkjlk test", Permissions: PermissionWriteRatings},
			nil,
			func(t *testing.T) {
				rdb.update = func(r *Role) error {
					cr := NewRole()
					cr.ID = 99
					cr.Label = "dfgsdflkjlkjlk test"
					cr.Permissions = PermissionWriteRatings

					assert.Equal(t, &cr, r)
					return nil
				}
			},
		},
		{
			"ok",
			&Role{ID: 99, Label: "    DFGSDFLKJlkjlk TEST   ", Permissions: PermissionWriteRatings},
			&Role{ID: 99, Label: "dfgsdflkjlkjlk test", Permissions: PermissionWriteRatings},
			nil,
			func(t *testing.T) {
				rdb.update = func(r *Role) error {
					cr := NewRole()
					cr.ID = 99
					cr.Label = "dfgsdflkjlkjlk test"
					cr.Permissions = PermissionWriteRatings

					assert.Equal(t, &cr, r)
					return nil
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			if cs.setup != nil {
				cs.setup(t)
			}

			err := rs.Update(cs.role)

			if cs.outerr != nil {
				assert.Error(t, err)
				assert.True(t, xerrors.Is(err, cs.outerr),
					"errors must match, expected %v, got %v", cs.outerr, err)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, cs.outrole, cs.role)
			}

			*rdb = testRoleDB{}
		})
	}
}

func TestRoleService_Delete(t *testing.T) {
	rdb := &testRoleDB{}
	rs := NewRoleService(nil)
	rs.(*roleService).RoleService.(*roleValidator).RoleDB = rdb // Mock

	t.Run("mustNotDeleteAdmin", func(t *testing.T) {
		rdb.delete = nil

		err := rs.Delete(1)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrReadOnly))
	})

	t.Run("mustNotDeleteUser", func(t *testing.T) {
		rdb.delete = nil

		err := rs.Delete(2)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrReadOnly))
	})

	t.Run("dbErrors", func(t *testing.T) {
		rdb.delete = func(id int64) error {
			assert.Equal(t, int64(888), id)
			return ErrNotFound
		}

		err := rs.Delete(888)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("ok", func(t *testing.T) {
		var called bool
		rdb.delete = func(id int64) error {
			assert.Equal(t, int64(888), id)
			called = true
			return nil
		}

		err := rs.Delete(888)

		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestRoleGORM_Create(t *testing.T) {
	var cases = []struct {
		name string
		role *Role

		outerr error

		setup func(t *testing.T, db *gorm.DB)
	}{
		{
			"repeatedID",
			&Role{ID: 99, Label: "test", Permissions: PermissionWriteRatings},
			ValidationError{"id": ErrIDTaken},
			func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&Role{ID: 99, Label: "other label", Permissions: 0}).Error)
			},
		},
		{
			"labelTaken",
			&Role{Label: "admin", Permissions: PermissionWriteRatings},
			ValidationError{"label": ErrDuplicate},
			nil,
		},
		{
			"internalError",
			&Role{Label: "accounts", Permissions: PermissionReadUsers},
			privateError("some internal error"),
			func(t *testing.T, db *gorm.DB) {
				dropRolesTable(db)
			},
		},
		{
			"ok",
			&Role{Label: "accounts", Permissions: PermissionReadUsers},
			nil,
			nil,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			db := setupGorm(t)

			if cs.setup != nil {
				cs.setup(t, db)
			}

			err := (&roleGorm{db}).Create(cs.role)

			if cs.outerr != nil {
				assert.Error(t, err)
				if _, ok := cs.outerr.(PublicError); ok {
					assert.True(t, xerrors.Is(err, cs.outerr))
				}

			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, int64(0), cs.role.ID, "must set the ID")
			}
		})
	}
}

func TestRoleGORM_Update(t *testing.T) {
	// Test Cases
	var cases = []struct {
		name   string
		role   *Role
		outerr error
		setup  func(t *testing.T, db *gorm.DB)
	}{
		{
			"idNotExists",
			&Role{ID: 99, Label: "accounts", Permissions: PermissionReadRatings},
			ErrNotFound,
			nil,
		},
		{
			"labelTaken",
			&Role{ID: 99, Label: "admin", Permissions: PermissionWriteRatings},
			ValidationError{"label": ErrDuplicate},
			func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&Role{ID: 99, Label: "tests", Permissions: PermissionWriteRatings}).Error)
			},
		},
		{
			"noChanges",
			&Role{ID: 99, Label: "tests", Permissions: PermissionWriteRatings},
			nil,
			func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&Role{ID: 99, Label: "tests", Permissions: PermissionWriteRatings}).Error)
			},
		},
		{
			"changesDefaultGoValues",
			&Role{ID: 99, Label: "", Permissions: 0},
			nil,
			func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&Role{ID: 99, Label: "tests", Permissions: PermissionWriteRatings}).Error)
			},
		},
		{
			"internalError",
			&Role{ID: 99, Label: "a test label", Permissions: PermissionReadRatings},
			privateError("any internal private error"),
			func(t *testing.T, db *gorm.DB) {
				dropRolesTable(db)
			},
		},
		{
			"ok",
			&Role{ID: 99, Label: "a test label", Permissions: PermissionReadRatings},
			nil,
			func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&Role{ID: 99, Label: "a test label", Permissions: PermissionWriteRatings}).Error)
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			db := setupGorm(t)

			if cs.setup != nil {
				cs.setup(t, db)
			}

			err := (&roleGorm{db}).Update(cs.role)

			if cs.outerr != nil {
				assert.Error(t, err)
				if _, ok := cs.outerr.(PublicError); ok {
					assert.True(t, xerrors.Is(err, cs.outerr))
				}

			} else {
				assert.NoError(t, err)

				var crole Role
				require.NoError(t, db.First(&crole, cs.role.ID).Error)
				assert.Equal(t, cs.role, &crole)
			}
		})
	}
}

func TestRoleGORM_Delete(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		err := (&roleGorm{db}).Delete(999)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropRolesTable(db)

		err := (&roleGorm{db}).Delete(999)

		assert.Error(t, err)
	})

	t.Run("inUse", func(t *testing.T) {
		db := setupGorm(t)
		user := &User{
			ID:       99,
			RoleID:   99,
			Active:   true,
			Email:    "test@test.com",
			Password: "TestPasswordHAsh",
		}
		role := &Role{
			ID:          99,
			Label:       "test",
			Permissions: 15,
		}

		require.NoError(t, db.Save(role).Error)
		require.NoError(t, db.Save(user).Error)

		err := (&roleGorm{db}).Delete(role.ID)
		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrInUse), "must indicate that the role is being used by an existing user")
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		role := &Role{
			ID:          99,
			Label:       "test",
			Permissions: 15,
		}

		require.NoError(t, db.Save(role).Error)

		err := (&roleGorm{db}).Delete(role.ID)
		assert.NoError(t, err)

		var ct int64
		assert.NoError(t, db.Model(&Role{}).Where("id = ?", 99).Count(&ct).Error)
		assert.Equal(t, int64(0), ct)
	})
}

func TestRoleGORM_ByID(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		_, err := (&roleGorm{db}).ByID(999)

		assert.Error(t, err)
		assert.True(t, xerrors.Is(err, ErrNotFound))
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropRolesTable(db)

		_, err := (&roleGorm{db}).ByID(999)

		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		role := &Role{
			ID:          99,
			Label:       "test",
			Permissions: 7,
		}

		require.NoError(t, db.Create(role).Error)

		outrole, err := (&roleGorm{db}).ByID(99)

		assert.NoError(t, err)
		assert.Equal(t, role, &outrole)
	})
}

func TestRoleGORM_ByIDs(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		db := setupGorm(t)

		users, err := (&roleGorm{db}).ByIDs(999)

		assert.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("otherErrors", func(t *testing.T) {
		db := setupGorm(t)
		dropRolesTable(db)

		_, err := (&roleGorm{db}).ByIDs(999)

		assert.Error(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		db := setupGorm(t)
		role1 := Role{
			ID:          99,
			Label:       "test",
			Permissions: 7,
		}
		role2 := Role{
			ID:          100,
			Label:       "atest",
			Permissions: 7,
		}

		require.NoError(t, db.Create(&role1).Error)
		require.NoError(t, db.Create(&role2).Error)

		t.Run("listAll", func(t *testing.T) {
			outrole, err := (&roleGorm{db}).ByIDs()

			assert.NoError(t, err)
			assert.Len(t, outrole, 4)
			assert.Contains(t, outrole, role1)
			assert.Contains(t, outrole, role2)
		})

		t.Run("listOne", func(t *testing.T) {
			outrole, err := (&roleGorm{db}).ByIDs(99)

			assert.NoError(t, err)
			assert.Len(t, outrole, 1)
			assert.Contains(t, outrole, role1)
		})

		t.Run("listOther", func(t *testing.T) {
			outrole, err := (&roleGorm{db}).ByIDs(100)

			assert.NoError(t, err)
			assert.Len(t, outrole, 1)
			assert.Contains(t, outrole, role2)
		})

		t.Run("listSome", func(t *testing.T) {
			outrole, err := (&roleGorm{db}).ByIDs(99, 100, 101, 102)

			assert.NoError(t, err)
			assert.Len(t, outrole, 2)
			assert.Contains(t, outrole, role1)
			assert.Contains(t, outrole, role2)
		})
	})
}
