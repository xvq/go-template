package job

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"log/slog"
	"time"
)

const ExampleType = "example:ping"

type ExamplePayload struct {
	Message string `json:"message"`
}

// Register is the only task registration entry point. No implicit init hooks.
func Register(log *slog.Logger) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(ExampleType, func(ctx context.Context, t *asynq.Task) error {
		var payload ExamplePayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("invalid example payload: %w", asynq.SkipRetry)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// Log metadata, not arbitrary payloads (which may contain secrets).
		log.InfoContext(ctx, "example task completed", "message_length", len(payload.Message), "completed_at", time.Now().UTC())
		return nil
	})
	return mux
}
