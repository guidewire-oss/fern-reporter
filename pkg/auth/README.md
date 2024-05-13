# Auth Package

The `auth` package provides middleware and utility functions for handling JWT-based authentication in web applications. 
It includes functionality for fetching, updating, and caching JSON Web Keys (JWKs) from a remote server and uses these keys to validate JWTs in HTTP requests.

## Features

- **Custom HTTP Client:** Configured to optionally skip TLS verification for development purposes.
- **JWKs Fetching and Caching:** Efficient management of fetching and caching JWKs for JWT validation.
- **JWT Validation Middleware:** Middleware that validates JWTs from the `Authorization` header of incoming HTTP requests using the cached JWKs.

## Components

### HTTP Client Configuration

The package defines a `customHTTPClient` which returns an HTTP client configured with a specific timeout and optional TLS settings.

```plaintext
function customHTTPClient
  return HTTP Client with:
    Timeout set to 30 seconds
    Optional TLS verification skipped
end function
```

### Fetching JWKs

The `fetchJWKs` function handles fetching JWKs from a provided URL, checking the HTTP response, and parsing the JWKs.

```plaintext
function fetchJWKs(url)
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

The `UpdateJWKs` function updates the cached JWK set using a new set fetched from a specified URL. This operation is thread-safe.

```plaintext
function UpdateJWKs(url)
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

The `getJWKs` function retrieves the cached JWK set, updating it if it's stale (more than an hour old).

```plaintext
function getJWKs(url)
  read lock on JWK cache
  if JWKs are older than 1 hour then
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

## Installation

- Assign an auth keys absolute URL here:`config/config.yaml` as a string. Under `auth`, there is a property 
labeled `keys-endpoint`.

- To integrate this package into your web application, ensure you have the necessary HTTP and JWT handling libraries installed, 
then import and configure this package.

## Security Notes

- **TLS Configuration:** Ensure TLS verification is enabled in production environments to protect against man-in-the-middle attacks.

This README provides a high-level overview and integration guide for the `auth` package, emphasizing secure and efficient JWT handling in web applications. Adjust the configurations and usage examples to suit your specific application requirements and environment.

