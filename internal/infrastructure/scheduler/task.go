package scheduler

import (
	"context"
	"time"
)

type Task struct {
	Name    string
	Spec    string
	Timeout time.Duration
	Run     func(context.Context) error
}
