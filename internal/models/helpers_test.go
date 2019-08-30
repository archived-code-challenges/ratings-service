package models

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "very lengthy jwt test secret to be used for tests"

var db *gorm.DB

// setupGorm helps initialising a GORM DB connection. It cleans up the database in each call.
func setupGorm(t *testing.T) *gorm.DB {
	dsl := os.Getenv("RATINGSAPP_POSTGRES_TEST_DSL")
	if dsl == "" {
		t.Skip("require RATINGSAPP_POSTGRES_TEST_DSL to run")
	}

	if db == nil {
		var err error
		db, err = gorm.Open("postgres", dsl)
		require.NoError(t, err)

		db.Exec("DROP SCHEMA public CASCADE")
		db.Exec("CREATE SCHEMA public")

		// uncoment this line to see helpful SQL output
		// db.LogMode(true)
	}

	err := db.DropTableIfExists(
		&User{},
		&Role{},
	).Error
	assert.NoError(t, err, "setupGorm: must drop existing tables")

	// use the services autoMigrate to fix up the DB
	err = (&Services{db: db}).autoMigrate()
	require.NoError(t, err)
	err = (&Services{db: db}).createDefaultValues()
	require.NoError(t, err)

	return db
}

func TestGORMTransaction(t *testing.T) {

	// we have some basic extra tests to gormTransaction here as some portions are not covered by
	// other tests of methods that use it.

	t.Run("cannotBegin", func(t *testing.T) {
		tdb := setupGorm(t)
		db = nil

		tdb.Close()
		err := gormTransaction(tdb, func(tx *gorm.DB) error {
			return nil
		})

		assert.Error(t, err, "must not begin a transaction if the database is closed")
	})

	t.Run("rollsBackOnPanic", func(t *testing.T) {
		db := setupGorm(t)

		assert.Panics(t, func() {
			gormTransaction(db, func(tx *gorm.DB) error {
				assert.NoError(t, tx.Create(&Role{ID: 99, Label: "test", Permissions: PermissionWriteRatings}).Error)
				panic("this transaction func has panicked")
			})
		}, "a panicked transation must re-raise the panic reason besides rolling back")

		var ct int64
		assert.NoError(t, db.Model(&Role{ID: 99}).Count(&ct).Error)
		assert.Equal(t, int64(0), ct)
	})

	t.Run("rollsBackOnError", func(t *testing.T) {
		db := setupGorm(t)

		err := gormTransaction(db, func(tx *gorm.DB) error {
			assert.NoError(t, tx.Create(&Role{ID: 99, Label: "test", Permissions: PermissionWriteRatings}).Error)
			return privateError("this is a test error")
		})

		assert.Error(t, err, "must return an error when the transation func returns an error")

		var ct int64
		assert.NoError(t, db.Model(&Role{ID: 99}).Count(&ct).Error)
		assert.Equal(t, int64(0), ct)
	})
}
