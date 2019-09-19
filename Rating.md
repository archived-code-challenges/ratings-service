Rating
======

- [Rating](#rating)
  - [Create](#create)
  - [List](#list)
  - [Get](#get)
  - [Update](#update)
  - [Delete](#delete)

A Rating resource represents an expression of value of any of the users of the system to a product, with a score and an optional commentary as well as other useful values described below.

The store of those ratings in addition to comments will include some useful data to check if the commentary is active or anonymous. A date keeping in the database the last (epoch) time of alteration of a rating, an 'extra' field to store information in json, a reference to the user to track who did a specific rating as the and finally, the rating score itself, that will be stored as an integer.

To make this API reusable and "end-user independent", the items will be stored with a target ID (product that is being rated) without creating a reference to a product in the database. Yet.

**Fields:**

| Field | Type | Default | Description |
| - | - | - | - |
| **id**        | int64     |       | Rating ID in the database. |
| **active**    | bool      | true  | Whether the rate is active. |
| **anonymous** | bool      | true  | Whether the rating is anonymous or not. |
| **comment**   | string    |       | The commentary attached to the rating. (max 255 characters) |
| **date**      | time.Time |       | Date when the rating was submitted or updated. |
| **extra**     | json      |  {}   | field to store stuff like logistics, color, date... in a json format. (max 255 characters) |
| **score**     | int       |       | Numeral value that will indicate the score that the target got in a rating. |
| **target**    | int64     |   *   | Numeral value that contains the target entity of the rating. |
| **userId**    | int64     |   **  | The ID of the user attached to this rating. |

*In a full adaptation of this API, the **target** can refer to a real object, on another table of the database. By now, we will treat all objects as a number for the sake of brevity.

**userId will be auto-assigned by the system to the user making the request, as denoted previously.

Functionality to have in mind:

- The pair (userId, target) needs to be unique in the system, this means one comment per user and target.
- A rating can only be updated by its owner.
- A rating can be deleted by its owner or by an administrator.
- Defaults for active, anonymous and extra would be applied if not supplied
- id, date and userId will be ignored if supplied.

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
}
```

**score** and **target** are mandatory.
The **active**, **anonymous**, **comment** and **extra** fields are optional, and the defaults apply if not supplied.

**date** and **userId** will be provided by the application.

**Response:**

```text
HTTP/1.1 201 Created
Content-Type: application/json

{
    "id": 99999,
    "active: true,
    "anonymous": true,
    "comment": "The article was amazing, but the case was a bit damaged.",
    "date": 1257894000,
    "extra" : {},
    "score": 4,
    "target": 1223456,
    "userId": 999,
}
```

Reponse codes:

* **201**: rating has been created.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **404**: User reference couldn't be found in the system using the session data.
* **409**: You are trying to duplicate an existing entity.

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
| comment must have max 255 characters | 400 | validation_error | comment: too_long |
| extra content is invalid | 400 | validation_error | extra: invalid |
| extra must have max 255 characters | 400 | validation_error | extra: too_long |
| score field is required | 400 | validation_error | score: required |
| target field is required | 400 | validation_error | target: required |
| target field is invalid | 400 | validation_error | target: invalid |
| userId field is invalid | 404 | validation_error | userId: reference_not_found |
| Input body is malformed | 400 | invalid_json | |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeRatings` permission | 403 | forbidden | |
| userId field is invalid | 404 | validation_error | userId: reference_not_found |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| ID field is invalid | 409 | validation_error | id: id_taken |
| target field for the given user already exists in the system | 409 | validation_error | target: is_duplicate |
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
    "active: true,
    "anonymous": true,
    "comment": "The article was amazing, but the case was a bit damaged.",
    "date": 1257894000,
    "extra" : {},
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
Remember, a rating can only be updated by its owner.

**Request:**

```text
PUT /api/v1/ratings/{id}
Content-Type: application/json

{
    "anonymous": false,
    "comment": "The article was exactly as I expected.",
    "extra": {"color":"blue"},
    "score": 9,
}

```

The **id** path parameter refers to the ID of the rating to be updated.

**score** is mandatory.
The **active**, **anonymous**, **comment** and **extra** fields are optional, and the defaults apply if not supplied.

**date**, **target** and **userId** will be provided by the application.

**Response:**

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
    "id": 999,
    "active: true,
    "anonymous": false,
    "comment": "The article was exactly as I expected.",
    "date": 1257894000,
    "extra": {"color":"blue"},
    "score": 9,
    "target": 1223456,
}
```

Reponse codes:

* **200**: Rating has been updated.
* **400**: The request could not be understood or has validation errors.
* **403**: The current user is not authorised to perform this operation.
* **404**: Requested ID not found or User reference couldn't be found in the system using the session data.
* **409**: You are trying to modify a read-only resource or trying to duplicate an existing entity.

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
  ]
}
```

| Case | HTTP code | error | fields |
| - | - | - | - |
| Input body is malformed | 400 | invalid_json | |
| comment must have max 255 characters | 400 | validation_error | comment: too_long |
| extra content is invalid | 400 | validation_error | extra: invalid |
| extra must have max 255 characters | 400 | validation_error | extra: too_long |
| score field is required | 400 | validation_error | score: required |
| Invalid Authorization header | 401 | unauthorised | |
| User does not have a `writeRatings` permission | 403 | forbidden | |
| userId field is invalid | 404 | validation_error | userId: reference_not_found |
| Invalid Content-Type/Accept, not wildcard or `application/json` | 406 | not_acceptable | |
| target field for the given user already exists in the system | 409 | validation_error | target: is_duplicate |
| User is not allowed to do the requested operation | 409 | read_only | |
| Internal error | 500 | server_error | |


Delete
------

Deletes a specific rating from the system.

A rating that has users attached cannot be deleted without first detaching the rating from them.

Administrators have the capability to delete any rating. The owner of a rating can remove it as long as it's logged in.

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
* **404**: Requested ID not found or User reference couldn't be found in the system using the session data.
* **409**: You are trying to delete a read-only resource.

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
| User is not allowed to do the requested operation | 409 | read_only | |
| Internal error | 500 | server_error | |
