package cli

import (
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/bootstrap"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/infrastructure/scheduler"
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
		r, err := bootstrap.Open(cmd.Context(), c, bootstrap.Options{Name: "schedule"})
		if err != nil {
			return err
		}
		defer r.Close()
		return scheduler.Run(cmd.Context(), c.Schedule, core.Log, schedule.Register())
	}}
}
