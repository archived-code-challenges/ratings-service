package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func filterItems(expected, actual map[string]interface{}) {
	eitm, ok := expected["items"]
	if !ok {
		return
	}

	aitm, ok := actual["items"]
	if !ok {
		return
	}

	expitems, ok := eitm.([]interface{})
	if !ok {
		return
	}

	actitems, ok := aitm.([]interface{})
	if !ok {
		return
	}

	if len(expitems) > len(actitems) {
		return
	}

	actual["items"] = actitems[:len(expitems)]
	actitems = actitems[:len(expitems)]

	for i, expitem := range expitems {
		actitem := actitems[i]

		expobj, ok := expitem.(map[string]interface{})
		if !ok {
			return
		}

		actobj, ok := actitem.(map[string]interface{})
		if !ok {
			return
		}

		for k := range actobj {
			if _, ok := expobj[k]; !ok {
				delete(actobj, k)
			}
		}
	}
}

// assertJSONSimilar does a similar comparision as assert.JSONEq, but it allows the test to pass if
// the JSON contents in "actual" are a superset of the expected contents.
func assertJSONSimilar(t *testing.T, expected, actual string, msgAndArgs ...interface{}) bool {
	var expectedJSONAsInterface, actualJSONAsInterface interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		return assert.Fail(t, fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()), msgAndArgs...)
	}

	if err := json.Unmarshal([]byte(actual), &actualJSONAsInterface); err != nil {
		return assert.Fail(t, fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", actual, err.Error()), msgAndArgs...)
	}

	switch exp := expectedJSONAsInterface.(type) {
	case map[string]interface{}:
		act, ok := actualJSONAsInterface.(map[string]interface{})
		if !ok {
			assert.Fail(t, fmt.Sprintf("Input ('%s') is not a JSON object.", actual), msgAndArgs...)
		}

		for k := range act {
			if _, ok := exp[k]; !ok {
				delete(act, k)
			}
		}

		// special treatment if it has an "items" element.
		filterItems(exp, act)
	}

	comparison := assert.Equal(t, expectedJSONAsInterface, actualJSONAsInterface, msgAndArgs...)

	if !comparison {
		assert.Nil(t, actual) // forces the wrong value to be printed
	}

	return comparison
}

func TestWebSever(t *testing.T) {
	type subCase struct {
		user           *testUser
		expectsCode    int
		expectsContent string
	}

	type mainCase struct {
		method string
		path   string
		input  string
		subs   []subCase
	}

	cases := []mainCase{
		// USERS
		{
			"POST",
			"/api/v1/users",
			`{"active":true,"email":"someone@some.com","firstName":"testname","lastName":"","password":"test1234","roleId":2}`,
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserWriteUsers, http.StatusCreated, `{"active":true,"email":"someone@some.com","firstName":"testname","lastName":"","roleId":2}`},
			},
		},
		{
			"GET",
			"/api/v1/users/7",
			"",
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusOK, `{"active":true,"email":"someone@some.com","firstName":"testname","lastName":"","roleId":2}`},
			},
		},
		{
			"GET",
			"/api/v1/users/?id=1,2",
			"",
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusOK, `{"items":[
					{"id":1,"active":true,"email":"admin@admin.com","firstName":"admin","lastName":"","roleId":1},
					{"id":2,"active":true,"email":"user@test.com","firstName":"user","lastName":"","roleId":2}
				]}`},
			},
		},
		{
			"PUT",
			"/api/v1/users/7",
			`{"active":true,"email":"someoneupdate@some.com","firstName":"readuser","lastName":"washere","password":"test1234","roleId":2}`,
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserWriteUsers, http.StatusOK, `{"active":true,"email":"someoneupdate@some.com","firstName":"readuser","lastName":"washere","roleId":2}`},
			},
		},
		{
			"DELETE",
			"/api/v1/users/7",
			"",
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserWriteUsers, http.StatusNoContent, ``},
			},
		},
		// ROLES
		{
			"POST",
			"/api/v1/roles",
			`{"label":"testrole","permissions":["readUsers","readRatings"]}`,
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserWriteUsers, http.StatusCreated, `{"label":"testrole","permissions":["readUsers","readRatings"]}`},
			},
		},
		{
			"GET",
			"/api/v1/roles/1",
			"",
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusOK, `{"id":1,"label":"admin","permissions":["readUsers","writeUsers","readRatings","writeRatings"]}`},
				{&testUserWriteUsers, http.StatusForbidden, `{"error":"forbidden"}`},
			},
		},
		{
			"GET",
			"/api/v1/roles/?id=1,2",
			"",
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusOK, `{
					"items":[
						{"id":1,"label":"admin","permissions":[
							"readUsers","writeUsers","readRatings","writeRatings"
						]},
						{"id":2,"label":"user","permissions":[]}
				]}`},
				{&testUserWriteUsers, http.StatusForbidden, `{"error":"forbidden"}`},
			},
		},
		{
			"PUT",
			"/api/v1/roles/7",
			`{"label":"testrole","permissions":["writeRatings"]}`,
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserWriteUsers, http.StatusOK, `{"label":"testrole","permissions":["writeRatings"]}`},
			},
		},
		{
			"DELETE",
			"/api/v1/roles/7",
			"",
			[]subCase{
				{&testUserNone, http.StatusUnauthorized, `{"error":"unauthorised"}`},
				{&testUserUser, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserReadUsers, http.StatusForbidden, `{"error":"forbidden"}`},
				{&testUserWriteUsers, http.StatusNoContent, ``},
			},
		},
	}

	for _, cs := range cases {
		for _, scs := range cs.subs {
			name := cs.method + ":" + cs.path + "(" + scs.user.name + ")"

			t.Run(name, func(t *testing.T) {
				req, _ := http.NewRequest(
					cs.method, testURL+cs.path, bytes.NewReader([]byte(cs.input)))
				req.Header.Add("Authorization", "Bearer "+scs.user.token)
				req.Header.Add("Content-Type", "application/json")

				res, err := http.DefaultClient.Do(req)
				require.NoError(t, err, "http client must not return any errors")

				b, _ := ioutil.ReadAll(res.Body)
				assert.Equal(t, scs.expectsCode, res.StatusCode)
				assert.Contains(t, res.Header.Get("Content-Type"), "application/json")

				if len(scs.expectsContent) > 0 {
					assertJSONSimilar(t, scs.expectsContent, string(b))
				} else {
					assert.Empty(t, string(b))
				}

			})
		}
	}
}
