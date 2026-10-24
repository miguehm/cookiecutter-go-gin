package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{ cookiecutter.go_module }}/internal/core/services"
	"{{ cookiecutter.go_module }}/internal/middlewares"
	"{{ cookiecutter.go_module }}/internal/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newTestApp(t *testing.T) (*gin.Engine, *testutil.FakeUserRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := testutil.NewConfig()
	repo := testutil.NewFakeUserRepository()
	controller := NewUserController(services.NewUserService(repo), cfg)

	app := gin.New()
	app.Use(middlewares.ErrorHandler(testutil.NewLogger()))
	app.POST("/signup", controller.CreateUser)
	app.POST("/login", controller.Login)
	app.GET("/me", middlewares.JwtAuth(cfg), controller.Me)

	return app, repo
}

func postJSON(app *gin.Engine, target, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	app.ServeHTTP(rec, req)
	return rec
}

func get(app *gin.Engine, target, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	app.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestCreateUserRejectsInvalidPayload(t *testing.T) {
	app, repo := newTestApp(t)

	rec := postJSON(app, "/signup", `{"email":"not-an-email"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400. body = %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	if body["code"] != "ErrBadRequest" {
		t.Errorf("code = %v", body["code"])
	}
	if len(repo.Users) != 0 {
		t.Error("no user must be created with an invalid payload")
	}
}

func TestCreateUserReturnsCreatedUserWithoutPassword(t *testing.T) {
	app, repo := newTestApp(t)

	payload := `{"firstname":"Ada","lastname":"Lovelace","email":"ada@example.com","password":"s3cret-password"}`
	rec := postJSON(app, "/signup", payload)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body = %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("body has no data: %s", rec.Body.String())
	}
	if data["email"] != "ada@example.com" {
		t.Errorf("email = %v", data["email"])
	}
	if strings.Contains(rec.Body.String(), "s3cret-password") ||
		strings.Contains(rec.Body.String(), `"password"`) {
		t.Fatalf("the response leaks the password: %s", rec.Body.String())
	}
	if len(repo.Users) != 1 {
		t.Fatalf("stored users = %d, want 1", len(repo.Users))
	}
}

func TestCreateUserRejectsDuplicatedEmail(t *testing.T) {
	app, repo := newTestApp(t)
	repo.CreateErr = gorm.ErrDuplicatedKey

	payload := `{"firstname":"Ada","lastname":"Lovelace","email":"ada@example.com","password":"s3cret-password"}`
	rec := postJSON(app, "/signup", payload)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409. body = %s", rec.Code, rec.Body.String())
	}
	if body := decode(t, rec); body["code"] != "ErrEmailTaken" {
		t.Errorf("code = %v", body["code"])
	}
}

func TestLoginAndMeFlow(t *testing.T) {
	app, repo := newTestApp(t)

	signup := postJSON(app, "/signup",
		`{"firstname":"Ada","lastname":"Lovelace","email":"ada@example.com","password":"s3cret-password"}`)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, body = %s", signup.Code, signup.Body.String())
	}

	login := postJSON(app, "/login", `{"email":"ada@example.com","password":"s3cret-password"}`)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200. body = %s", login.Code, login.Body.String())
	}
	data, _ := decode(t, login)["data"].(map[string]any)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("login did not return a token: %s", login.Body.String())
	}

	me := get(app, "/me", token)
	if me.Code != http.StatusOK {
		t.Fatalf("GET /me status = %d, want 200. body = %s", me.Code, me.Body.String())
	}
	meData, _ := decode(t, me)["data"].(map[string]any)
	if meData["email"] != "ada@example.com" {
		t.Errorf("email = %v", meData["email"])
	}

	for _, stored := range repo.Users {
		if stored.Password == "s3cret-password" {
			t.Fatal("the password must be stored hashed")
		}
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	app, _ := newTestApp(t)

	postJSON(app, "/signup",
		`{"firstname":"Ada","lastname":"Lovelace","email":"ada@example.com","password":"s3cret-password"}`)

	rec := postJSON(app, "/login", `{"email":"ada@example.com","password":"wrong-password"}`)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401. body = %s", rec.Code, rec.Body.String())
	}
	if body := decode(t, rec); body["code"] != "ErrInvalidCredentials" {
		t.Errorf("code = %v", body["code"])
	}
}

func TestLoginRejectsUnknownUser(t *testing.T) {
	app, _ := newTestApp(t)

	rec := postJSON(app, "/login", `{"email":"ghost@example.com","password":"s3cret-password"}`)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401. body = %s", rec.Code, rec.Body.String())
	}
}

func TestMeRequiresToken(t *testing.T) {
	app, _ := newTestApp(t)

	rec := get(app, "/me", "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestMeRejectsInvalidToken(t *testing.T) {
	app, _ := newTestApp(t)

	rec := get(app, "/me", "not-a-jwt")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestLoginRejectsInvalidPayload(t *testing.T) {
	app, _ := newTestApp(t)

	rec := postJSON(app, "/login", `{"email":"ada@example.com"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400. body = %s", rec.Code, rec.Body.String())
	}
}
