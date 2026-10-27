package job

import "github.com/hibiken/asynq"

// Register is the only task registration entry point. No implicit init hooks.
func Register() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(ExampleType, handleExample)
	return mux
}
