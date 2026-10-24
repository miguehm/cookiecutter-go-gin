package apierrors

import "net/http"

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

func (e APIError) WithErr(err error) *APIError {
	e.Err = err
	return &e
}

var (
	ErrApikeyInvalid = &APIError{
		Status:    http.StatusUnauthorized,
		ErrorCode: "ErrApikeyInvalid",
		Message:   "Api key invalid",
	}

	ErrAccessTokenInvalid = &APIError{
		Status:    http.StatusUnauthorized,
		ErrorCode: "ErrAccessTokenInvalid",
		Message:   "Access token invalid",
	}

	ErrInvalidCredentials = &APIError{
		Status:    http.StatusUnauthorized,
		ErrorCode: "ErrInvalidCredentials",
		Message:   "Invalid email or password",
	}

	ErrUserNotFound = &APIError{
		Status:    http.StatusNotFound,
		ErrorCode: "ErrUserNotFound",
		Message:   "User not found",
	}

	ErrEmailTaken = &APIError{
		Status:    http.StatusConflict,
		ErrorCode: "ErrEmailTaken",
		Message:   "Email already registered",
	}
)
