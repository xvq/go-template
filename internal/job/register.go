package job

import "github.com/hibiken/asynq"

// Register is the only task registration entry point. No implicit init hooks.
func Register() (*asynq.ServeMux, int) {
	tasks := []struct {
		name    string
		handler asynq.HandlerFunc
	}{
		{ExampleType, handleExample},
	}
	mux := asynq.NewServeMux()
	for _, task := range tasks {
		mux.Handle(task.name, task.handler)
	}
	return mux, len(tasks)
}
