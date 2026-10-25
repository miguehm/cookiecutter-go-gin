package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// TypeEmailDelivery is the queue name of the email delivery task.
const TypeEmailDelivery = "email:deliver"

// EmailDeliveryPayload is the payload carried by the email delivery task.
type EmailDeliveryPayload struct {
	UserID     int
	TemplateID string
}

// HandleEmailDelivery unmarshals the payload and performs the delivery,
// ignoring the context.
func HandleEmailDelivery(_ context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %w: %w", err, asynq.SkipRetry)
	}
	log.Printf("Sending Email to User: user_id=%d, template_id=%s", p.UserID, p.TemplateID)
	// Email delivery code ...
	return nil
}
