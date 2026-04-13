package client

import "fmt"

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("tracker API request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("tracker API request failed with status %d: %s", e.StatusCode, e.Message)
}

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type AuthError struct{ Message string }

func (e *AuthError) Error() string { return e.Message }

type NotFoundError struct{ Message string }

func (e *NotFoundError) Error() string { return e.Message }

type NetworkError struct{ Err error }

func (e *NetworkError) Error() string { return e.Err.Error() }

func (e *NetworkError) Unwrap() error { return e.Err }
