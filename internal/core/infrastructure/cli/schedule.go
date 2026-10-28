package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/scheduler"
	"github.com/xvq/go-template/internal/schedule"
)

func newSchedule(path *string) *cobra.Command {
	return &cobra.Command{Use: "schedule", Short: "启动定时调度（单实例）", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		if err := c.Validate("schedule"); err != nil {
			return err
		}
		r, err := bootstrap.Open(cmd.Context(), c, bootstrap.ScheduleOptions())
		if err != nil {
			return err
		}
		defer r.Close()
		tasks := schedule.Register()
		return scheduler.Run(cmd.Context(), c, core.Log, tasks, func() {
			fmt.Fprintf(cmd.ErrOrStderr(), "Scheduler started, registered tasks: %d\n", len(tasks))
		})
	}}
}
