package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"{{ cookiecutter.go_module }}/internal/testutil"
)

func TestUserControllerSignUp(t *testing.T) {
	t.Parallel()

	controller := UserController{}
	ctx, recorder := testutil.NewContext(http.MethodGet, "/signup", nil)

	controller.SignUp(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("response does not contain data: %s", recorder.Body.String())
	}
}
