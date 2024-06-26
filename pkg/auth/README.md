# Auth Package

The `auth` package provides middleware and utility functions for handling JWT-based authentication in web applications. 
It includes functionality for fetching, updating, and caching JSON Web Keys (JWKs) from a remote server and uses these keys to validate JWTs in HTTP requests.

## Features

- **JWKs Fetching and Caching:** Efficient management of fetching and caching JWKs for JWT validation.
- **JWT Validation Middleware:** Middleware that validates JWTs from the `Authorization` header of incoming HTTP requests using the cached JWKs.
- **Offline Validation:** Since the JWKs are cached, validation can be performed offline.

## Components

### HTTP Client Configuration

The package defines a `customHTTPClient` which returns an HTTP client configured with a specific timeout and optional TLS settings.
**This is temporary.**

```plaintext
function customHTTPClient
  return HTTP Client with:
    Timeout set to 30 seconds
    Optional TLS verification skipped
end function
```

### Fetching JWKs

The `fetchJWKS` function handles fetching JWKs from a provided URL, checking the HTTP response, and parsing the JWKs.

```plaintext
function fetchJWKS(url)
  create HTTP client
  send GET request to url
  if response is OK then
    read response body
    parse JWKs from response data
    return JWKs
  else
    return error
  end if
end function
```

### Updating JWKs

The `UpdateJWKS` function updates the cached JWK set using a new set fetched from a specified URL. This operation is thread-safe.

```plaintext
function UpdateJWKS(url)
  fetch JWKs from url
  if fetch successful then
    lock JWK cache
    update cached JWK set
    update last updated timestamp
    unlock JWK cache
    return success
  else
    return fetch error
  end if
end function
```

### Retrieving JWKs with Caching

The `getJWKS` function retrieves the cached JWK set, updating it if it's stale (more than an 12 hours old).

```plaintext
function getJWKS(url)
  read lock on JWK cache
  if JWKs are older than 12 hours then
    update JWKs
  end if
  return cached JWKs
end function
```


### JWT Authentication Middleware

This middleware function validates JWTs by extracting them from the `Authorization` header, verifying them against the cached JWKs, and storing the result in the context.

```plaintext
middleware JWTAuthMiddleware(url)
  use getJWKs to fetch current JWK set
  if error fetching JWKs then
    respond with error
    stop processing request
  end if
  extract JWT from Authorization header
  if JWT is valid then
    store validated token in context
    continue processing request
  else
    respond with unauthorized error
    stop processing request
  end if
end middleware
```

## Usage

- Assign an auth keys absolute URL here:`config/config.yaml` as a string. Under `auth`, there is a property 
labeled `keys-endpoint`. Alternatively, you could set the auth keys absolute URL in this environment variable: `AUTH_KEYS_ENDPOINT`.

## Security Notes

- **TLS Configuration:** Ensure TLS verification is enabled in production environments.