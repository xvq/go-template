package cli

import (
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/core/bootstrap"
	"github.com/xvq/go-template/internal/core/config"
)

func newServer(path *string) *cobra.Command {
	return &cobra.Command{Use: "server", Short: "启动 HTTP API", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load(*path)
		if err != nil {
			return err
		}
		if err := c.Validate("server"); err != nil {
			return err
		}
		return bootstrap.RunServer(cmd.Context(), c)
	}}
}
