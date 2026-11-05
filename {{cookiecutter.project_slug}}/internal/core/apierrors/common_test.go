package apierrors

import (
	"errors"
	"net/http"
	"testing"
)

func TestSentinelErrorsHaveStableStatusAndCode(t *testing.T) {
	cases := []struct {
		name string
		err  *APIError
		want int
		code string
	}	{
		{"apikey", ErrApikeyInvalid, http.StatusUnauthorized, "ErrApikeyInvalid"},
		{"token", ErrAccessTokenInvalid, http.StatusUnauthorized, "ErrAccessTokenInvalid"},
		{"credentials", ErrInvalidCredentials, http.StatusUnauthorized, "ErrInvalidCredentials"},
		{"notfound", ErrUserNotFound, http.StatusNotFound, "ErrUserNotFound"},
		{"taken", ErrEmailTaken, http.StatusConflict, "ErrEmailTaken"},
	}
	for _, tc := range cases {
		if tc.err.Status != tc.want {
			t.Errorf("%s: Status = %d, want %d", tc.name, tc.err.Status, tc.want)
		}
		if tc.err.ErrorCode != tc.code {
			t.Errorf("%s: ErrorCode = %s, want %s", tc.name, tc.err.ErrorCode, tc.code)
		}
	}
}

func TestAPIErrorText(t *testing.T) {
	bare := APIError{Message: "User not found"}
	if bare.Error() != "User not found" {
		t.Errorf("bare Error() = %q", bare.Error())
	}
	wrapped := bare
	wrapped.Err = errors.New("no rows in result set")
	if got := wrapped.Error(); got != "User not found:\n\tno rows in result set" {
		t.Errorf("wrapped Error() = %q", got)
	}
}

func TestWithErrReturnsACopy(t *testing.T) {
	cause := errors.New("duplicate key")
	out := ErrEmailTaken.WithErr(cause)

	if out.Err == nil {
		t.Fatal("WithErr must attach the cause")
	}
	if !errors.Is(out.Err, cause) {
		t.Errorf("attached err = %v", out.Err)
	}
	if ErrEmailTaken.Err != nil {
		t.Error("the original sentinel must stay cause-free")
	}
	if out == ErrEmailTaken {
		t.Error("WithErr must return a copy, not the same pointer")
	}
	if out.Message != ErrEmailTaken.Message || out.Status != ErrEmailTaken.Status {
		t.Error("the copy must preserve status and message")
	}
}
