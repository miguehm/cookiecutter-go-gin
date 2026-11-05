package asynctasks

import (
	"strings"
	"testing"

	"{{ cookiecutter.go_module }}/internal/core/asynctasks/tasks"
)

type unmarshalable struct {
	Ch chan int
}

func TestNewTaskMarshalsPayload(t *testing.T) {
	task, err := NewTask("demo:type", map[string]any{"user_id": 7})
	if err != nil {
		t.Fatalf("NewTask returned error: %v", err)
	}
	if task.Type() != "demo:type" {
		t.Errorf("type = %s", task.Type())
	}
	if got := string(task.Payload()); !strings.Contains(got, `"user_id":7`) {
		t.Errorf("payload = %s", got)
	}
}

func TestNewTaskRejectsUnmarshalablePayload(t *testing.T) {
	if _, err := NewTask("demo:type", unmarshalable{Ch: make(chan int)}); err == nil {
		t.Fatal("expected an error for an unmarshalable payload")
	} else if !strings.Contains(err.Error(), "chan") {
		t.Errorf("error = %v", err)
	}
}

func TestPublishWithoutClient(t *testing.T) {
	var nilPublisher *Publisher
	err := nilPublisher.Publish("demo:type", map[string]any{})
	if err == nil {
		t.Fatal("nil publisher must error")
	} else if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error = %v", err)
	}

	publisher := NewPublisher(nil)
	err = publisher.Publish("demo:type", map[string]any{})
	if err == nil {
		t.Fatal("nil client must error")
	} else if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error = %v", err)
	}
}

func TestDebugTaskWithoutClientErrorsCleanly(t *testing.T) {
	publisher := NewPublisher(nil)
	if err := publisher.DebugTask(); err == nil {
		t.Fatal("expected an error without a client")
	}
}

func TestSendEmailBuildsTheDemoTaskShape(t *testing.T) {
	task, err := NewTask(tasks.TypeEmailDelivery, map[string]any{"user_id": 1, "template_id": "welcome-email"})
	if err != nil {
		t.Fatalf("NewTask returned error: %v", err)
	}
	if task.Type() != "email:deliver" {
		t.Errorf("type = %s", task.Type())
	}
}
