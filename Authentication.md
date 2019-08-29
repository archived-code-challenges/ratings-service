Authentication
==============

- [Authentication](#authentication)
  - [With password](#with-password)
  - [With refresh token](#with-refresh-token)

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
