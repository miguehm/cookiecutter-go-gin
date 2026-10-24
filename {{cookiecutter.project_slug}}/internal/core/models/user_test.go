package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
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

func TestUserPrimaryKeyIsUUID(t *testing.T) {
	field, ok := reflect.TypeOf(User{}).FieldByName("ID")
	if !ok {
		t.Fatal("User has no ID field")
	}
	if field.Type != reflect.TypeOf(uuid.UUID{}) {
		t.Fatalf("ID type = %s, want uuid.UUID", field.Type)
	}
	tag := field.Tag.Get("gorm")
	for _, part := range []string{"primaryKey", "type:uuid", "default:gen_random_uuid()"} {
		if !strings.Contains(tag, part) {
			t.Errorf("gorm tag %q must contain %q", tag, part)
		}
	}
}
