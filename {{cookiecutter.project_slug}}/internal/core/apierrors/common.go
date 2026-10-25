// Package apierrors declares the errors that are rendered verbatim by the
// error handler, each carrying its HTTP status and error code.
package apierrors

import "net/http"

// APIError is an error with an HTTP status, a stable error code and a
// user-facing message.
type APIError struct {
	Status    int
	ErrorCode string
	Message   string
	Err       error
}

func (e APIError) Error() string {
	if e.Err != nil {
		return e.Message + ":\n	" + e.Err.Error()
	}
	return e.Message
}

// WithErr returns a copy of e wrapping err, for logging the cause without
// exposing it to the client.
func (e APIError) WithErr(err error) *APIError {
	e.Err = err
	return &e
}

var (
	// ErrApikeyInvalid reports an unknown or malformed API key.
	ErrApikeyInvalid = &APIError{
		Status:    http.StatusUnauthorized,
		ErrorCode: "ErrApikeyInvalid",
		Message:   "Api key invalid",
	}

	// ErrAccessTokenInvalid reports a malformed or expired JWT.
	ErrAccessTokenInvalid = &APIError{
		Status:    http.StatusUnauthorized,
		ErrorCode: "ErrAccessTokenInvalid",
		Message:   "Access token invalid",
	}

	// ErrInvalidCredentials is returned when the email or the password does
	// not match.
	ErrInvalidCredentials = &APIError{
		Status:    http.StatusUnauthorized,
		ErrorCode: "ErrInvalidCredentials",
		Message:   "Invalid email or password",
	}

	// ErrUserNotFound is returned when no user matches the given ID.
	ErrUserNotFound = &APIError{
		Status:    http.StatusNotFound,
		ErrorCode: "ErrUserNotFound",
		Message:   "User not found",
	}

	// ErrEmailTaken is returned when the email is already registered.
	ErrEmailTaken = &APIError{
		Status:    http.StatusConflict,
		ErrorCode: "ErrEmailTaken",
		Message:   "Email already registered",
	}
)
