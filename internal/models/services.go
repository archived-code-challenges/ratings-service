/*
Package models implements persistence/fetch/update and validation for application structs.
*/
package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // gorm's postgres support
)

// Services aggregates all services provided by the models package.
type Services struct {
	User   UserService
	Role   RoleService
	Rating RatingService

	db *gorm.DB
}

// Config defines configuration options for instantiating new Services values.
type Config struct {
	// JWTSecret is used to sign the JWT tokens used to
	// identify users.
	JWTSecret []byte

	// DatabaseDSL is the database connection string to
	// the database. Currently, it only accepts Postgres.
	DatabaseDSL string
}

// NewServices instantiate and configures a new Services value.
func NewServices(c *Config) (*Services, error) {
	var s Services

	err := c.check()
	if err != nil {
		return nil, wrap("invalid configuration", err)
	}

	s.db, err = gorm.Open("postgres", c.DatabaseDSL)
	if err != nil {
		return nil, wrap("failed to connect to postgres", err)
	}

	err = s.autoMigrate()
	if err != nil {
		return nil, wrap("can't migrate", err)
	}

	s.Role = NewRoleService(s.db)

	s.User, err = NewUserService(s.db, s.Role, c.JWTSecret)
	if err != nil {
		return nil, wrap("can't start UserService", err)
	}

	s.Rating = NewRatingService(s.db)

	err = s.createDefaultValues()
	if err != nil {
		return nil, wrap("can't insert default values", err)
	}

	return &s, nil
}

// Close release all resources related to s.
func (s *Services) Close() error {
	err := s.db.Close()
	if err != nil {
		return wrap("failed to close database connections", err)
	}

	return nil
}

// autoMigrate creates or updates the database schema.
func (s *Services) autoMigrate() error {
	err := s.db.
		AutoMigrate(&Role{}).
		AutoMigrate(&User{}).AddForeignKey("role_id", "roles(id)", "RESTRICT", "RESTRICT").
		AutoMigrate(&Rating{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").
		Error
	if err != nil {
		return err
	}

	return nil
}

// createDefaultValues uses the instantiated services to add the set of fixed, default
// values that the database is supposed to have.
func (s *Services) createDefaultValues() error {
	// warning: non standard SQL used to update sequence counters
	err := s.db.
		Save(&Role{ID: 1, Label: "admin", Permissions: Permissions(-1)}).
		Save(&Role{ID: 2, Label: "user", Permissions: Permissions(0)}).
		Exec("DO $$ BEGIN IF (SELECT last_value = 1 FROM roles_id_seq) THEN ALTER SEQUENCE roles_id_seq RESTART WITH 3; END IF; END; $$").
		Error
	if err != nil {
		return wrapi("failed to create default values when migrating", err)
	}

	err = s.db.Save(&User{ID: 1, Active: true, Email: "admin@admin.com", FirstName: "admin", Password: "$2y$12$5wXQu8UknGQxEvdATbjvUORLJAQXYfB7tLCqqISFZqjlXz3f9FYwO", RoleID: 1}).
		Exec("DO $$ BEGIN IF (SELECT last_value = 1 FROM users_id_seq) THEN ALTER SEQUENCE users_id_seq RESTART WITH 2; END IF; END; $$").
		Error
	if err != nil {
		return wrapi("failed to create default roles when migrating", err)
	}

	return nil
}

// check does basic verification on configuration keys. It will not
// verify values that would otherwise result in a failed initialisation of
// the services.
func (c *Config) check() error {
	if len(c.JWTSecret) < 32 {
		return ErrJWTSecretTooShort
	}

	return nil
}
