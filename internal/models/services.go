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
	// TODO: Implement auto-migrations here.

	return nil
}

func (s *Services) createDefaultValues() error {
	// TODO: Implement defaults here

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
