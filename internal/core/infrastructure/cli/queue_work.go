package cli

import (
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/taskqueue"
	"github.com/xvq/go-template/internal/job"
)

func newQueueWork(path *string) *cobra.Command {
	return &cobra.Command{Use: "queue:work", Short: "启动 Asynq 队列消费者", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		r, err := bootstrap.Open(cmd.Context(), c, bootstrap.Options{Name: "queue:work", Queue: true})
		if err != nil {
			return err
		}
		defer r.Close()
		srv := taskqueue.NewServer(c, core.Log)
		if err = srv.Start(job.Register()); err != nil {
			return err
		}
		<-cmd.Context().Done()
		srv.Shutdown()
		return nil
	}}
}
