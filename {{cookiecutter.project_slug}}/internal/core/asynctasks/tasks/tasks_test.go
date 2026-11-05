package tasks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
)

const (
	typeDebug = TypeDebugTask
	typeEmail = TypeEmailDelivery
)

func marshalPayload(t *testing.T, payload any) []byte {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("payload does not marshal: %v", err)
	}
	return raw
}

func TestHandleDebugTaskAcceptsPayload(t *testing.T) {
	raw := marshalPayload(t, DebugTaskPayload{UserID: 1, Message: "hello"})
	if err := HandleDebugTask(context.Background(), asynq.NewTask(typeDebug, raw)); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

func TestHandleDebugTaskRejectsMalformedPayload(t *testing.T) {
	if err := HandleDebugTask(context.Background(), asynq.NewTask(typeDebug, []byte("not-json"))); err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
}

func TestHandleEmailDeliveryAcceptsPayload(t *testing.T) {
	raw := marshalPayload(t, EmailDeliveryPayload{UserID: 2, TemplateID: "welcome-email"})
	if err := HandleEmailDelivery(context.Background(), asynq.NewTask(typeEmail, raw)); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

func TestHandleEmailDeliveryRejectsMalformedPayload(t *testing.T) {
	if err := HandleEmailDelivery(context.Background(), asynq.NewTask(typeEmail, []byte("["))); err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
}
