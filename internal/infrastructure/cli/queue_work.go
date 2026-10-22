package cli

import (
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/bootstrap"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/infrastructure/taskqueue"
	"github.com/xvq/go-template/internal/job"
)

func newQueueWork(path *string) *cobra.Command {
	return &cobra.Command{Use: "queue:work", Short: "启动 Asynq 队列消费者", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		r, err := bootstrap.Open(cmd.Context(), c, "queue-work", "queue")
		if err != nil {
			return err
		}
		defer r.Close()
		srv := taskqueue.NewServer(c, r.Log)
		if err = srv.Start(job.Register(r.Log)); err != nil {
			return err
		}
		<-cmd.Context().Done()
		srv.Shutdown()
		return nil
	}}
}
