package models

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func dropUsersTable(db *gorm.DB) {
	db.DropTableIfExists(&User{})
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
