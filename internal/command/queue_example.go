package command

import (
	"context"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/job"
	"time"
)

func newQueueExample(path *string) *cobra.Command {
	var delay time.Duration
	cmd := &cobra.Command{Use: "queue:example", Short: "投递无副作用的示例任务", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if delay < 0 {
			return fmt.Errorf("delay 不能为负数")
		}
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		r, err := bootstrap.Open(cmd.Context(), c, bootstrap.Options{Name: "queue:example", Queue: true})
		if err != nil {
			return err
		}
		defer r.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		info, err := core.Queue.Enqueue(ctx, job.ExampleType, job.ExamplePayload{Message: "hello"}, asynq.ProcessIn(delay))
		if err != nil {
			return err
		}
		cmd.Printf("任务已入队: %s\n", info.ID)
		return nil
	}}
	cmd.Flags().DurationVar(&delay, "delay", 0, "延迟时间，例如 10s")
	return cmd
}
