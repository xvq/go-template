package bootstrap

import "github.com/xvq/go-template/internal/core/config"

// ServerOptions selects HTTP API resources, enabling queue access by configuration.
func ServerOptions(c config.Config) Options {
	return Options{
		Name:     "server",
		Database: true,
		Redis:    true,
		HTTP:     false,
		Queue:    c.Queue.Enabled,
	}
}

// ScheduleOptions selects resources used by the registered scheduled tasks.
func ScheduleOptions() Options {
	return Options{Name: "schedule"}
}

// QueueWorkOptions selects resources used by the queue consumer and its jobs.
func QueueWorkOptions() Options {
	return Options{Name: "queue:work", Queue: true}
}
