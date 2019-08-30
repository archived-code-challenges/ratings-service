package models

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServices(t *testing.T) {

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

}
