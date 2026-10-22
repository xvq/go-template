package schedule

import (
	"github.com/xvq/go-template/internal/infrastructure/scheduler"
	"log/slog"
)

// Register lists scheduled tasks explicitly. Define each task in its own file
// and add its constructor here. Accept dependencies as parameters when needed.
func Register(log *slog.Logger) []scheduler.Task {
	return []scheduler.Task{
		newHeartbeat(log),
	}
}
