package command

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/service/settings"
)

func newSettingsClearCache(path *string) *cobra.Command {
	return &cobra.Command{Use: "settings:clear-cache", Short: "清空设置缓存", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		r, err := bootstrap.Open(ctx, c, bootstrap.Options{Name: "settings:clear-cache", Redis: true})
		if err != nil {
			return err
		}
		defer r.Close()
		if err := settings.ClearCache(ctx); err != nil {
			return err
		}
		cmd.Println("设置缓存已清空")
		return nil
	}}
}
