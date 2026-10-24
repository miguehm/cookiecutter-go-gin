package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserJSONOmitsPassword(t *testing.T) {
	user := User{Password: "super-secret-hash"}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if strings.Contains(string(data), "super-secret-hash") {
		t.Fatalf("password leaked in JSON: %s", data)
	}
	if strings.Contains(string(data), `"password"`) {
		t.Fatalf("password field leaked in JSON: %s", data)
	}
}

func TestUserJSONKeepsPublicFields(t *testing.T) {
	user := User{
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Email:     "ada@example.com",
		Password:  "super-secret-hash",
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	for _, expected := range []string{`"firstname":"Ada"`, `"lastname":"Lovelace"`, `"email":"ada@example.com"`} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("expected %s in JSON: %s", expected, data)
		}
	}
}
