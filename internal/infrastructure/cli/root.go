package cli

import (
	"github.com/spf13/cobra"
	"github.com/xvq/go-template/internal/command"
)

func NewRoot() *cobra.Command {
	var path string
	root := &cobra.Command{Use: "app", Short: "Go API 模板", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVarP(&path, "config", "c", "config.yaml", "配置文件路径")
	root.AddCommand(
		newConfigInit(&path),
		newServer(&path),
		newSchedule(&path),
		newQueueWork(&path),
		newMigrate(&path),
	)
	root.AddCommand(command.Register(&path)...)
	return root
}
