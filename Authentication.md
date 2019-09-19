Authentication
==============

- [Authentication](#authentication)
  - [With password](#with-password)
  - [With refresh token](#with-refresh-token)
- [User](#user)
  - [Create](#create)
  - [List](#list)
  - [Get](#get)
  - [Update](#update)
  - [Delete](#delete)
- [Role](#role)
  - [Create](#create-1)
  - [List](#list-1)
  - [Get](#get-1)
  - [Update](#update-1)
  - [Delete](#delete-1)

RatingAPI's authentication is a subset of the OAuth 2.0 standard, where the password and refresh token grant types are used to obtain access to an access and a refresh token.

The refresh token is used to obtain a new access token after expiration without asking the user again for username and password. When a refresh token expires, a 401 error is returned to indicate it cannot be used to obtain a new access token, and username and password are the only options to retrieve a new access token.

RatingAPI uses email addresses as usernames, simplifying what users need to memorise and also facilitating future social login implementation.


With password
-------------

The request must be sent form-encoded, and the response will be sent JSON encoded.

**Request:**

```text
POST /api/v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=password
&email=user@example.com
&password=1234secret
```

Parameters:

* **grant_type**: Must be "password".
* **email**: RatingAPI user's email address.
* **password**: RatingAPI user's password.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "access_token":"MTQ0NjJkZmQ5OTM2NDE1ZTZjNGZmZjI3",
  "token_type":"bearer",
  "expires_in":21600,
  "refresh_token":"IwOGYzYTlmM2YxOTQ5MGE3YmNmMDFkNTVk"
}
```

Values:

* **access_token**: Access token, used to interact with the API.
* **expires_in**: Duration of the access token in seconds.
* **refresh_token**: Refresh token.
* **token_type**: Will always be "bearer".

Errors:

* **200**: User is authorised, tokens included.
* **400**: The request could not be understood or is malformed.
* **401**: Client authentication failed.

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "invalid_request",
  "error_description": "Request missing 'password' field"
}
```

| Case | HTTP code | error | error_description |
| - | - | - | - |
| Invalid Content-Type, is not `application/x-www-form-urlencoded` | 400 | invalid_request | content_type_not_accepted |
| Body is not properly encoded as a form | 400 | invalid_request | invalid_form |
| Unsupported value in `grant_type` | 400 | unsupported_grant_type | |
| Internal error | 500 | server_error | |
| Credentials are empty | 400 | invalid_request | credentials_not_provided |
| Credentials are not found or not accepted | 401 | invalid_client | |

**Find out more:** [Authentication concept](https://developer.okta.com/docs/concepts/authentication/); [Password grant](https://www.oauth.com/oauth2-servers/access-tokens/password-grant/); [OAuth response](https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/)


With refresh token
------------------

The request must be sent form-encoded, and the response will be sent JSON encoded.

A token request using a refresh token will return a new, current access token as well as a new refresh token, extending the lifetime of the user session and reducing chances of the user needing to login again to the system, as long as the user access the system frequently.

**Request:**

```text
POST /api/v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=IwOGYzYTlmM2YxOTQ5MGE3YmNmMDFkNTVk
```

Parameters:

* **grant_type**: Must be "refresh_token".
* **refresh_token**: A previously issued refresh_token.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "access_token":"MTQ0NjJkZmQ5OTM2NDE1ZTZjNGZmZjI3",
  "token_type":"bearer",
  "expires_in":21600,
  "refresh_token":"IwOGYzYTlmM2YxOTQ5MGE3YmNmMDFkNTVk"
}
```

Values:

* **access_token**: Access token, used to interact with the API.
* **expires_in**: Duration of the access token in seconds.
* **refresh_token**: Refresh token.
* **token_type**: Will always be "bearer".

Errors:

* **200**: User is authorised, tokens are included.
* **400**: The request could not be understood or is malformed.
* **401**: Client authentication failed.

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "invalid_request",
  "error_description": "Request missing 'refresh_token' field"
}
```

| Case | HTTP code | error | error_description |
| - | - | - | - |
| Invalid Content-Type, is not `application/x-www-form-urlencoded` | 400 | invalid_request | content_type_not_accepted |
| Body is not properly encoded as a form | 400 | invalid_request | invalid_form |
| Unsupported value in `grant_type` | 400 | unsupported_grant_type | |
| Internal error | 500 | server_error | |
| Refresh token is empty | 400 | invalid_request | credentials_not_provided |
| Refresh token's user not found or not accepted | 401 | invalid_client | |

**Find out more:** [Refresh token grant](https://www.oauth.com/oauth2-servers/access-tokens/refreshing-access-tokens/); [OAuth response](https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/)


User
====

A **User** resource represents a human user of the system. Users are attached to roles, which determine what they are allowed to perform on the system. A user must always have a role, and is created with a default "user" role that is allowed basic read access to some parts of the application.

A user that has the admin role can only have its role changed by another user with admin role.

**Fields:**

| Field | Type | Default | Description |
| - | - | - | - |
| **id** | int | | User ID in the database. |
| **active** | bool | true | Whether the account is active. An inactive account is not able to login to the application, or perform any actions via the API. |
| **email** | string |  | User email address. Used for user identification, login. Must be unique in the application. |
| **firstName**, **lastName** | string |  | User name details. The first name is mandatory. |
| **password** | string |  | User password. Must be passed on create/update operations. It's never returned on any read operations |
| **roleId** | int64 | `user` role ID | The ID of the role attached to this user. The default is the `user` role (2), which results in minimum read-only permissions. |


Create
------

Performs the creation of new users.

The email address must be unique in the system. The password must have more than 8 characters.

**Request:**

```text
POST /api/v1/users/
Content-Type: application/json

{
    "active": true,
    "email": "rick@sanchez.com",
    "firstName": "Rick",
    "lastName": "Sanchez",
    "password": "RickdiculouslyEasy1234",
    "roleId": 99
}
```

The **active**, **, **roleId** and **lastName** are optional, and the defaults apply if not supplied. The **email** and **password** values are ignored if is set to true.

**Response:**

```
HTTP/1.1 201 Created
Content-Type: application/json

{
    "id": 990,
    "active": true,
    "email": "rick@sanchez.com",
    "firstName": "Rick",
    "lastName": "Sanchez",
    "roleId": 99
}
```

Reponse codes:

* **201**: User has been created.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": {
      "password"
      "roleId"
      "email"
  }
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeUsers` permission | 403 | forbidden | |
| Internal error | 500 | server_error | |
| Input body is malformed | 400 | invalid_json | |
| Email address is taken | 409 | validation_error | email: email_taken |
| Email address is empty | 400 | validation_error | email: email_not_provided |
| Email address is invalid | 400 | validation_error | email: invalid_email_address |
| First name is empty | 400 | validation_error | firstName: first_name_not_provided |
| First name is too short | 400 | validation_error | firstName: first_name_too_short |
| Password is empty | 400 | validation_error | password: password_not_provided |
| Password is too short | 400 | validation_error | password: password_too_short |
| Role ID does not exist | 400 | validation_error | roleId: role_id_not_found |


List
----

Returns a list with all users in the system, or a subset of them.

**Request:**

```text
GET /api/v1/users/?id=999,888
```

The **id** query parameter is an optional comma separated list of IDs. Items that do not exist will silently be left out of the returned list.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json
{
    "items": [
        {
            "id": 999,
            "active": true,
            "email": "rick@sanchez.com",
            "firstName": "Rick",
            "lastName": "Sanchez",
            "roleId": 99
        },
        {
            "id": 888,
            "active": true,
            "email": "someoneelse@somewhere.com",
            "firstName": "Adam",
            "lastName": "Doe",
            "roleId": 99
        }
    ]
}
```

Reponse codes:

* **200**: Request completed successfully.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": {
      "id": "invalid_parse"
  }
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `readUsers` permission | 403 | forbidden | |
| Internal error | 500 | server_error | |
| Query parameter `id` is malformed | 400 | validation_error | id: invalid_parse |


Get
---

Returns a specific user's details.

**Request:**

```text
GET /api/v1/users/{id}
```

The **id** path parameter refers to the ID of the user to be returned.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json
{
    "id": 999,
    "active": true,
    "email": "rick@sanchez.com",
    "firstName": "Rick",
    "lastName": "Sanchez",
    "roleId": 99
}
```

The password field is not returned in this case.

Reponse codes:

* **200**: Request completed successfully.

Error example:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "not_found"
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `readUsers` permission | 403 | forbidden | |
| Internal error | 500 | server_error | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |


Update
------

Updates an existing user.

The email address must be unique in the system. The password must have more than 8 characters.

**Request:**

```text
PUT /api/v1/users/{id}
Content-Type: application/json

{
    "active": true,
    "email": "rick@sanchez.com",
    "firstName": "Rick",
    "lastName": "Sanchez",
    "password": "RickdiculouslyEasy1234",
    "roleId": 99
}
```

The **id** path parameter refers to the ID of the user to be updated.

The **active**, **roleId**, **lastName** and **password** are optional, and the defaults apply if not supplied. Notice that if any of these fields is not provided, the defaults __WILL BE WRITTEN__ to the updated user.

The **password** is optional and the previous password is kept if it is not provided or an empty string is set.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 990,
    "active": true,
    "email": "rick@sanchez.com",
    "firstName": "Rick",
    "lastName": "Sanchez",
    "roleId": 99
}
```

Reponse codes:

* **200**: User has been updated.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **409**: A user with the same email address already exists with another ID.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": [
      "password",
      "roleId",
      "email"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeUsers` permission | 403 | forbidden | |
| Internal error | 500 | server_error | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |
| Item refers to the default admin user | 409 | read_only | |
| Input body is malformed | 400 | invalid_json | |
| Email address is taken | 409 | validation_error | email: email_taken |
| Email address is empty | 400 | validation_error | email: email_not_provided |
| Email address is invalid | 400 | validation_error | email: invalid_email_address |
| First name is empty | 400 | validation_error | firstName: first_name_not_provided |
| First name is too short | 400 | validation_error | firstName: first_name_too_short |
| Password is too short | 400 | validation_error | password: password_too_short |
| Role ID does not exist | 400 | validation_error | roleId: role_id_not_found |


Delete
------

Deletes a specific user from the system.

**Request:**

```text
DELETE /api/v1/users/{id}
```

The **id** path parameter refers to the ID of the user to be deleted.

**Response:**

```text
HTTP/1.1 204 No Content
Content-Type: application/json

```

Reponse codes:

* **204**: Request completed successfully.

Error example:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "not_found"
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeUsers` permission | 403 | forbidden | |
| Internal error | 500 | server_error | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |
| Item refers to the default admin user | 409 | read_only | |


Role
====

A **Role** resource represents a set of permissions that can be attached to users, allowing or preventing them to perform specific operations on the system. Not all parts of the system have a permission attached, which means that any user will have access to operations on that part of the system.

If a permission exists for a specific operation or part of the system, then users will need to be in a role that contains the permission to work on the specific part of the system.

The admin role is a special role that cannot be modified.

Users that have no role attached still have minimum read-only access to the system. The minimum read-only access excludes all parts of the system that require a permission to read or write.

These are the current permissions available:

| ID | Go enum | Effect |
| - | - | - |
| readUsers | PermissionReadUsers | Allows reading and listing users and roles |
| writeUsers | PermissionWriteUsers | Allows creating, updating and deleting users and roles |
| readRatings| PermissionReadRatings| Allows reading rating elements. |
| writeRatings| PermissionWriteRatings| Allows creating, updating and deleting rating elements. |

The way this works is everything that does NOT have a read permission, is allowed to be read by anyone, and everything that does NOT have a write permission, is allowed to be written by anyone.
Therefore, the only things that need permission to be read are users. Everything else can be read by anyone with any role or any set of permissions.

**Fields:**

| Field | Type | Default | Description |
| - | - | - | - |
| **id** | int | | Role ID in the database. |
| **label** | string |  | The role's label as a friendly name. The minimum length required is 4 characters. |
| **permissions** | []string | [] | User email address. Used for user identification, login. Must be unique in the application. |


Create
------

Performs the creation of new roles.

The label must be unique in the system and have a minimum of 4 characters.


**Request:**

```text
POST /api/v1/roles/
Content-Type: application/json

{
    "label": "Workers",
    "permissions": [
        "readUsers",
        "readRatings"
    ]
}
```

The **permissions** field is optional, and the defaults apply if not supplied.

**Response:**

```text
HTTP/1.1 201 Created
Content-Type: application/json

{
    "id": 990,
    "label": "Workers",
    "permissions": [
        "readUsers",
        "readRatings"
    ]
}
```

Reponse codes:

* **201**: Role has been created.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **409**: A role with the same label already exists.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": [
      "label"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Input body is malformed | 400 | invalid_json | |
| label field is empty | 400 | validation_error | label: label_not_provided |
| Label must have at least 4 characters | 400 | validation_error | label: label_too_short |
| Invalid Authorization header | 401 | unauthorised | |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| User does not have a `writeUsers` permission | 403 | forbidden | |
| Label is taken | 409 | validation_error | label: label_taken |
| Primary key already exists | 409 | validation_error | id: id_taken |
| Internal error | 500 | server_error | |


List
----

Returns a list of all roles in the system.

**Request:**

```text
GET /api/v1/roles/?id=999,888
```

The **id** query parameter is an optional comma separated list of IDs. Items that do not exist will silently be left out of the returned list.

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "items": [
        {
            "id": 990,
            "label": "Workers",
            "permissions": [
                "readUsers",
                "readRatings"
            ]
        },
        {
            "id": 991,
            "label": "Agronomists",
            "permissions": [
                "writeRatings"
            ]
        }
    ]
}
```

Reponse codes:

* **200**: Request completed successfully.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "invalid_query_param",
  "fields": [
      "id"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Query parameter `id` is malformed | 400 | validation_error | id: invalid_parse |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `readUsers` permission | 403 | forbidden | |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Internal error | 500 | server_error | |


Get
---

Returns a specific role's details.

**Request:**

```text
GET /api/v1/roles/{id}
```

The **id** path parameter refers to the ID of the role to be returned.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 990,
    "label": "Workers",
    "permissions": [
        "readUsers",
        "readRatings"
    ]
}
```

Reponse codes:

* **200**: Request completed successfully.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **404**: Requested ID not found.

Error example:

```text
HTTP/1.1 404 Not Found
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "not_found"
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `readUsers` permission | 403 | forbidden | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Internal error | 500 | server_error | |


Update
------

Updates an existing role.

The label must be unique in the system and have a minimum of 4 characters.

**Request:**

```text
PUT /api/v1/roles/{id}
Content-Type: application/json

{
    "label": "Workers",
    "permissions": [
        "readUsers",
        "readRatings"
    ]
}
```

The **id** path parameter refers to the ID of the role to be updated.

All fields are mandatory.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 990,
    "label": "Workers",
    "permissions": [
        "readUsers",
        "readRatings"
    ]
}
```

Reponse codes:

* **200**: Role has been updated.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **409**: A role with the same label already exists with another ID.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": [
      "label"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Input body is malformed | 400 | invalid_json | |
| label field is empty | 400 | validation_error | label: label_not_provided |
| Label must have at least 4 characters | 400 | validation_error | label: label_too_short |
| Invalid Authorization header | 401 | unauthorised | |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| User does not have a `writeUsers` permission | 403 | forbidden | |
| Label is taken | 409 | validation_error | label: label_taken |
| Primary key already exists | 409 | validation_error | id: id_taken |
| Resource cannot be modified or deleted | 409 | validation_error | id: read_only |
| Resource cannot be modified or deleted | 409 | validation_error | label: read_only |
| Internal error | 500 | server_error | |


Delete
------

Deletes a specific role from the system.

A role that has users attached cannot be deleted without first detaching the role from them.

**Request:**

```text
DELETE /api/v1/roles/{id}
```

The **id** path parameter refers to the ID of the role to be deleted.

**Response:**

```text
HTTP/1.1 204 No Content
Content-Type: application/json

```

Reponse codes:

* **204**: Request completed successfully.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **404**: Requested ID not found.
* **409**: There are users still attached to the role.

Error example:

```text
HTTP/1.1 409 Conflict
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "role_in_use",
  "users": [
    990, 991, 992
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeUsers` permission | 403 | forbidden | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Resource cannot be deleted because other resources depend on it | 409 | in_use | |
| Resource cannot be modified or deleted | 409 | validation_error | id: read_only |
| Resource cannot be modified or deleted | 409 | validation_error | label: read_only |
| Internal error | 500 | server_error | |
