package schedule

import (
	"github.com/xvq/go-template/internal/core/infrastructure/scheduler"
)

// Register lists scheduled tasks explicitly. Define each task in its own file
// and add its constructor here. Tasks access infrastructure through core.
func Register() []scheduler.Task {
	return []scheduler.Task{
		newHeartbeat(),
	}
}
