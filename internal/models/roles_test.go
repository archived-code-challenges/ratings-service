package models

import (
	"encoding/json"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func dropRolesTable(db *gorm.DB) {
	db.DropTableIfExists(&Role{})
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
