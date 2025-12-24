package main

import "errors"

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")

	ErrMissingJWT = errors.New("missing JWT")
	ErrInvalidJWT = errors.New("invalid JWT")
	ErrFailedToGetJWTSecret = errors.New("failed to get JWT secret")
)
