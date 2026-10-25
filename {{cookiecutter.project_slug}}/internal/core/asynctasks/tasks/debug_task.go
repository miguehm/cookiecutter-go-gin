// Package tasks holds the Asynq task definitions, their payloads and the
// handlers executed by the worker.
package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// TypeDebugTask is the queue name of the debug task.
const TypeDebugTask = "debug:task"

// DebugTaskPayload is the payload carried by the debug task.
type DebugTaskPayload struct {
	UserID  int
	Message string
}

// HandleDebugTask unmarshals the payload and logs it, ignoring the context.
func HandleDebugTask(_ context.Context, t *asynq.Task) error {
	var p DebugTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %w: %w", err, asynq.SkipRetry)
	}
	log.Printf("Running debug task: user_id=%d, message=%s", p.UserID, p.Message)
	return nil
}
