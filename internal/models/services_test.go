package models

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServices(t *testing.T) {

	testServices := func(t *testing.T, services *Services) {
		roles, err := services.Role.ByIDs()
		assert.NoError(t, err, "basic test on roles does not return errors")

		users, err := services.User.ByIDs()
		assert.NoError(t, err, "basic test on users does not return errors")

		require.Len(t, roles, 2, "must create default roles")
		assert.Equal(t, int64(1), roles[0].ID)
		assert.Equal(t, "admin", roles[0].Label)
		assert.Equal(t, int64(2), roles[1].ID)
		assert.Equal(t, "user", roles[1].Label)

		newRole := Role{Label: "test label"}
		assert.NoError(t, services.Role.Create(&newRole), "must update sequence numbers in postgres so new roles can be created")
		services.Role.Delete(newRole.ID)

		require.Len(t, users, 1, "must create default user")
		assert.Equal(t, int64(1), users[0].ID)
		assert.Equal(t, "admin", users[0].FirstName)

		newUser := User{Active: true, Email: "test@server.com", FirstName: "test", Password: "very long password", RoleID: 1}
		assert.NoError(t, services.User.Create(&newUser), "must update sequence numbers in postgres so new users can be created")
		services.User.Delete(newUser.ID)

	}

	t.Run("invalidConfig1", func(t *testing.T) {
		c := Config{
			JWTSecret: []byte("test secret"),
		}
		_, err := NewServices(&c)

		assert.Error(t, err, "must not allow invalid configuration")
	})

	t.Run("invalidConfig2", func(t *testing.T) {
		c := Config{
			JWTSecret: []byte("test secret with long size and other"),
		}
		_, err := NewServices(&c)

		assert.Error(t, err, "must not allow missing database DSL")
	})

	t.Run("invalidConfig3", func(t *testing.T) {
		c := Config{
			JWTSecret:   []byte("test secret with long size and other"),
			DatabaseDSL: "postgres://user:password@host:5433/database?sslmode=disable",
		}
		_, err := NewServices(&c)

		assert.Error(t, err, "must not create service if it fails to connect to the database")
	})

	t.Run("okNewDatabase", func(t *testing.T) {
		dsl := os.Getenv("RATINGSAPP_POSTGRES_TEST_DSL")
		if dsl == "" {
			t.Skip("require RATINGSAPP_POSTGRES_TEST_DSL to run")
		}

		db, err := gorm.Open("postgres", dsl)
		require.NoError(t, err)
		db.Exec("DROP SCHEMA public CASCADE")
		db.Exec("CREATE SCHEMA public")
		db.Close()

		c := Config{
			JWTSecret:   []byte("test secret with long size and other"),
			DatabaseDSL: dsl,
		}
		services, err := NewServices(&c)
		assert.NoError(t, err, "must not create service if it fails to connect to the database")

		testServices(t, services)
	})

	t.Run("ok", func(t *testing.T) {
		dsl := os.Getenv("RATINGSAPP_POSTGRES_TEST_DSL")
		if dsl == "" {
			t.Skip("require RATINGSAPP_POSTGRES_TEST_DSL to run")
		}

		c := Config{
			JWTSecret:   []byte("test secret with long size and other"),
			DatabaseDSL: dsl,
		}
		services, err := NewServices(&c)
		assert.NoError(t, err, "must not create service if it fails to connect to the database")

		testServices(t, services)
	})
}

func TestServices_Close(t *testing.T) {
	dsl := os.Getenv("RATINGSAPP_POSTGRES_TEST_DSL")
	if dsl == "" {
		t.Skip("require RATINGSAPP_POSTGRES_TEST_DSL to run")
	}

	c := Config{
		JWTSecret:   []byte("test secret with long size and other"),
		DatabaseDSL: dsl,
	}
	services, err := NewServices(&c)
	assert.NoError(t, err, "must not create service if it fails to connect to the database")

	assert.NoError(t, services.Close())

	_, err = services.Role.ByIDs()
	assert.Error(t, err, "basic test on a closed service for roles must return an error")

	_, err = services.User.ByIDs()
	assert.Error(t, err, "basic test on a closed service for users must return an error")

}
