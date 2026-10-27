package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/xvq/go-template/internal/core"
)

const ExampleType = "example:ping"

type ExamplePayload struct {
	Message string `json:"message"`
}

func handleExample(ctx context.Context, t *asynq.Task) error {
	var payload ExamplePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("invalid example payload: %w", asynq.SkipRetry)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Log metadata, not arbitrary payloads (which may contain secrets).
	core.Log.InfoContext(ctx, "example task completed", "message_length", len(payload.Message), "completed_at", time.Now().UTC())
	return nil
}
