Rating
======

- [Rating](#rating)
  - [Create](#create)
  - [List](#list)
  - [Get](#get)
  - [Update](#update)
  - [Delete](#delete)

A Rating resource represents an expression of value of any of the users in the system of a product, that will be referred as an ID, without a reference in the database.

The store of those ratings will include comments, a reference to the user to track who did an specific comment and the rate itself, that will be stored as an integer.

To make this API reusable and "end-user independent", the items will be stored with a product ID (product that is being rated)

**Fields:**

| Field | Type | Default | Description |
| - | - | - | - |
| **id**        | int       |       | Rating ID in the database. |
| **active**    | bool      | true  | Whether the rate is active. |
| **anonymous** | bool      | true  | Whether the rating is anonymous or not. |
| **comment**   | string    |       | The commentary attached to the rating. |
| **date**      | time.Time |       | Date when the rating was submitted or updated. |
| **extra**     | json      |  {}   | field to store stuff like logistics, color, date... in a json format. |
| **score**     | int       |       | Numeral value that will indicate the score that the target got in a rating. |
| **target***   | int64     |       | Numeral value that contains the target entity of the rating. |
| **userId**    | int64     |       | The ID of the user attached to this rating. |

*In a full adaptation of this API, the **target** can refer to a real object, on another table of the database. By now, we will treat all objects as a number for the sake of brevity.

The pair (userId, target) needs to be unique in the system // One comment per user.

Create
------

Performs the creation of new ratings.

**Request:**

```text
POST /api/v1/ratings/
Content-Type: application/json

{
    "comment": "The article was amazing, but the case was a bit damaged.",
    "score": 4,
    "target": 9999555,
    "userId": 999
}
```

The **active**, **anonymous** and **extra** fields are optional, and the defaults apply if not supplied.

**Response:**

```text
HTTP/1.1 201 Created
Content-Type: application/json

{
    "id": 99999,
    "comment": "The article was amazing, but the case was a bit damaged.",
    "date": 1257894000,
    "score": 4,
    "target": 1223456,
    "userId": 999,
}
```

Reponse codes:

* **201**: rating has been created.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **409**: A rating with the same pair: target, userId already exists.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": [
        "comment",
        "extra",
        "score",
        "target"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Input body is malformed | 400 | invalid_json | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeRatings` permission | 403 | forbidden | |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| ID field is invalid | 400 | validation_error | id: id_taken |
| comment field is empty | 400 | validation_error | comment: required |
| comment must have at least 2 characters | 400 | validation_error | name: too_short |
| comment must have max 255 characters | 400 | validation_error | name: too_long |
| extra field is invalid | 400 | validation_error | extra: invalid |
| extra must have max 255 characters | 400 | validation_error | name: too_long |
| score field is invalid | 400 | validation_error | score: invalid |
| target field is invalid | 400 | validation_error | target: invalid |
| userId field is invalid | 400 | validation_error | userId: reference_not_found |
| Internal error | 500 | server_error | |


List
----

Returns a list of all ratings in the system.

**Request:**

```text
GET /api/v1/ratings/?target=999
```

The **target** query parameter target is optional. A successfull result will be a list of all the ratings attached to a specific target.

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "items": [
        {
            "id": 88888,
            "comment": "The article was amazing, but the case was a bit damaged.",
            "date": 1257894000,
            "score": 4,
            "target": 1223456,
            "userId": 444
        },
        {
            "id": 99999,
            "comment": "The article was slightly better that I expected.",
            "date": 1257894000,
            "score": 5,
            "target": 1223456,
            "userId": 555
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
      "target"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Query parameter target is malformed | 400 | validation_error | id: invalid_query_param |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `readRatings` permission | 403 | forbidden | |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Internal error | 500 | server_error | |


Get
---

Returns a specific rating's details.

**Request:**

```text
GET /api/v1/ratings/{id}
```

The **id** path parameter refers to the ID of the rating to be returned.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 99999,
    "comment": "The article was amazing, but the case was a bit damaged.",
    "date": 1257894000,
    "score": 4,
    "target": 1223456,
    "userId": 999
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
| User does not have a `readRatings` permission | 403 | forbidden | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Internal error | 500 | server_error | |


Update
------

Updates an existing rating.

**Request:**

```text
PUT /api/v1/ratings/{id}
Content-Type: application/json

{
    "anonymous": true,
    "comment": "The article was exactly as I expected.",
    "extra": {"color":"blue"},
    "score": 9,
}

```

The **id** path parameter refers to the ID of the rating to be updated.

All fields are mandatory.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 99999,
    "anonymous": true,
    "comment": "The article was exactly as I expected.",
    "date": 1257894000,
    "extra": {"color":"blue"},
    "score": 9,
    "target": 1223456,
    "userId": 999,
}
```

Reponse codes:

* **200**: Rating has been updated.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **409**: You are trying to modify a read-only resource.

Error example:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
  "error": "validation_error",
  "fields": [
        "date",
        "extra",
        "target",
        "userId"
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Input body is malformed | 400 | invalid_json | |
| comment must have at least 2 characters | 400 | validation_error | name: too_short |
| Invalid Authorization header | 401 | unauthorised | |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| User does not have a `writeRatings` permission | 403 | forbidden | |
| Primary key already exists | 409 | validation_error | id: id_taken |
| Resource cannot be modified or deleted | 409 | validation_error | id: read_only |
| Resource cannot be modified or deleted | 409 | validation_error | userId: read_only |
| Resource cannot be modified or deleted | 409 | validation_error | target: read_only |
| Resource cannot be modified or deleted | 409 | validation_error | date: read_only |
| Internal error | 500 | server_error | |


Delete
------

Deletes a specific rating from the system.

A rating that has users attached cannot be deleted without first detaching the rating from them.

**Request:**

```text
DELETE /api/v1/ratings/{id}
```

The **id** path parameter refers to the ID of the rating to be deleted.

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
| User does not have a `writeRatings` permission | 403 | forbidden | |
| Path parameter `id` is not an integer | 404 | not_found | |
| Item could not be found | 404 | not_found | |
| Invalid Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| Resource cannot be modified or deleted | 409 | validation_error | id: read_only |
| Internal error | 500 | server_error | |
