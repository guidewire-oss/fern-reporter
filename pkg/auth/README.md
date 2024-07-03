# Auth Middleware
This repository provides JWT authentication and scope-based authorization middleware for the Gin web framework. 
It leverages the lestrrat-go/jwx library for handling JWT tokens and JSON Web Key Sets (JWKS).

## Features
- **JWKs Fetching and Caching:** Efficient management of fetching and caching JWKs for JWT validation.
- **JWT Validation Middleware:** Middleware that validates JWTs from the `Authorization` header of incoming HTTP requests using the cached JWKs.
- **Offline Validation:** Since the JWKs are cached, validation can be performed offline.
- **Scope Middleware:**  Middleware to check user permissions based on token scopes.

## Configuration
You can load configuration values using the `config.yaml` or environment variables.

### Environment Variables
Ensure the following environment variables are set or set a default values in `config.yaml`:
- `SCOPE_CLAIM_NAME`: Name of the claim used for scopes.
- `AUTH_JSON_WEB_KEYS_ENDPOINT`: URL of the JWKS endpoint.

### Configuration Files
Load necessary configurations using `config.yaml`.

## Middleware Setup

### JWT Middleware
- Fetches JWKS from the specified URL.
- Validates the JWT token present in the Authorization header.
- Extracts and validates the scope claim from the token.

### Scope Middleware
- Checks if the user has the required permissions based on the scope extracted from the JWT token.

## Usage
To use the middleware, import the package and apply the middleware to your Gin router. 
Ensure the necessary environment variables and configurations are set before running the server.