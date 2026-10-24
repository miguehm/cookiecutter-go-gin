package asynctasks

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"{{ cookiecutter.go_module }}/internal/core/asynctasks/tasks"
)

// Publisher enqueues tasks through an injected Asynq client. It replaces the
// previous package level helpers that read the client from a global.
type Publisher struct {
	client *asynq.Client
}

func NewPublisher(client *asynq.Client) *Publisher {
	return &Publisher{client: client}
}

// Publish enqueues a task built from an arbitrary payload.
func (p *Publisher) Publish(taskType string, payload any, opts ...asynq.Option) error {
	task, err := NewTask(taskType, payload)
	if err != nil {
		return fmt.Errorf("could not create task: %w", err)
	}
	if p == nil || p.client == nil {
		return fmt.Errorf("asynq client is not initialized")
	}
	if _, err := p.client.Enqueue(task, opts...); err != nil {
		return fmt.Errorf("could not enqueue task: %w", err)
	}
	return nil
}

func (p *Publisher) DebugTask() error {
	return p.Publish(
		tasks.TypeDebugTask,
		tasks.DebugTaskPayload{UserID: 1, Message: "hello"},
		asynq.Retention(24*time.Hour), asynq.Timeout(3*time.Second),
	)
}

func (p *Publisher) SendEmail() error {
	return p.Publish(
		tasks.TypeEmailDelivery,
		tasks.EmailDeliveryPayload{UserID: 1, TemplateID: "welcome-email"},
		asynq.Retention(24*time.Hour), asynq.Timeout(3*time.Second),
	)
}
