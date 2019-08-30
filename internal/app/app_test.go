package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

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

var (
	testUserNone         testUser
	testUserAdmin        testUser
	testUserUser         testUser
	testUserReadUsers    testUser
	testUserWriteUsers   testUser
	testUserReadRatings  testUser
	testUserWriteRatings testUser
)

type testUser struct {
	// name for a testUser is either the permission the user has, or "admin"
	// for the admin user, or "user" for a user without permissions.
	name string

	// token is the auth token to be used when impersonating this user.
	token string
}

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
	createUsersAndRoles()

	// runs the tests
	flag.Parse()
	ret := m.Run()

	// tear down the application
	app.Shutdown()
	os.Exit(ret)
}

func isServerUp() {
	var err error
	for i := 6; i > 0; i-- {
		_, err = http.Post(testURL+"/api/v1/oauth/token", "application/x-www-form-urlencoded", nil)
		if err == nil {
			return
		}

		time.Sleep(1 * time.Second)
	}

	panic(xerrors.Errorf("attempted to connect to test app, did not succeed: %w", err))
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

func createUsersAndRoles() {
	var tok, utok string
	var rid int64

	testUserNone.name = "none"

	// for the admin user, we just get a token
	tok = getAuthToken("admin@admin.com", "password")
	testUserAdmin.name = "admin"
	testUserAdmin.token = tok

	// for the "user" user, the role is already there, so we create the user and
	// get the token
	testUserUser.name = "user"
	createUser(tok, "user", 2)
	utok = getAuthToken("user@test.com", "password")
	testUserUser.token = utok

	// for all other users, we create both a role and a user with the same name
	testUserReadUsers.name = "readUsers"
	testUserWriteUsers.name = "writeUsers"
	testUserReadRatings.name = "readRatings"
	testUserWriteRatings.name = "writeRatings"

	for _, tu := range []*testUser{
		&testUserReadUsers,
		&testUserWriteUsers,
		&testUserReadRatings,
		&testUserWriteRatings,
	} {
		rid = createRole(tok, tu.name)
		createUser(tok, tu.name, rid)
		utok = getAuthToken(tu.name+"@test.com", "password")
		tu.token = utok
	}

}

func getAuthToken(username, password string) string {
	data := url.Values{
		"grant_type": []string{"password"},
		"email":      []string{username},
		"password":   []string{password},
	}

	res, err := http.Post(testURL+"/api/v1/oauth/token", "application/x-www-form-urlencoded",
		bytes.NewReader([]byte(data.Encode())))
	if err != nil {
		panic(xerrors.Errorf("failed to authenticate %s: %w", username, err))

	}

	if res.StatusCode != http.StatusOK {
		panic(xerrors.Errorf("autenticating %s must return status code 200", username))
	}

	var response map[string]interface{}
	json.NewDecoder(res.Body).Decode(&response)

	return response["access_token"].(string)
}

// createRole creates a role with name <name> and returns its ID. The role will have a single
// permission that is equal to the name on its JSON encoding.
func createRole(token, name string) int64 {
	req, _ := http.NewRequest(
		http.MethodPost,
		testURL+"/api/v1/roles/",
		bytes.NewReader([]byte(`{"label": "`+name+`","permissions": ["`+name+`"]}`)),
	)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(xerrors.Errorf("failed to create role %s: %w", name, err))
	}

	if res.StatusCode != http.StatusCreated {
		panic(xerrors.Errorf("creating role %s must return status code 201", name))
	}

	var response map[string]interface{}
	json.NewDecoder(res.Body).Decode(&response)

	return int64(response["id"].(float64))
}

// createUser creates a new user with email <name>@test.com and password "password".
func createUser(token, name string, roleID int64) {
	req, _ := http.NewRequest(
		http.MethodPost, testURL+"/api/v1/users/",
		bytes.NewReader([]byte(`{
			"email":"`+name+`@test.com",
			"firstName":"`+name+`",
			"password":"password",
			"roleId":`+strconv.FormatInt(roleID, 10)+`}`),
		),
	)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(xerrors.Errorf("failed to create user %s: %w", name, err))
	}

	if res.StatusCode != http.StatusCreated {
		panic(xerrors.Errorf("creating user %s must return status code 201", name))
	}
}
