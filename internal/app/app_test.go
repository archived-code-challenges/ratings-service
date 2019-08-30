package app

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"golang.org/x/xerrors"
)

// The tests in the app package are meant to be simple checks that cover if:
//
//	- auth requirements are being applied
//	- all routes are correctly wired and poiting to the controllers
//
// As such, it won't extensively test every corner case, but mostly that things
// are in a basic working state when app runs.

const (
	testJWTSecret = "very lengthy jwt test secret to be used for tests"
	testPort      = "59999"
	testURL       = "http://localhost:59999"
)

var app App

func TestMain(m *testing.M) {
	// setup app
	dsl := os.Getenv("RATINGSAPP_POSTGRES_TEST_DSL")
	if dsl == "" {
		fmt.Println("SKIP: require RATINGSAPP_POSTGRES_TEST_DSL to run")
		os.Exit(0)
	}

	// clean up the database
	cleanUpDatabase(dsl)

	app.Configure(&Config{
		DSL:       dsl,
		Port:      testPort,
		JWTSecret: testJWTSecret,
	})

	// start running the full application
	go func() {
		app.Run()
	}()

	// prepare things
	isServerUp()
	createDefaultSchema()

	// runs the tests
	flag.Parse()
	ret := m.Run()

	// tear down the application
	app.Shutdown()
	os.Exit(ret)
}

func isServerUp() {
	// TODO: To be implemented
}

func cleanUpDatabase(dsl string) {
	db, err := gorm.Open("postgres", dsl)
	if err != nil {
		panic(xerrors.Errorf("failed to connect to the database: %w", err))
	}

	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")
	db.Close()
}

func createDefaultSchema() {
	// TODO: To be implemented
}
