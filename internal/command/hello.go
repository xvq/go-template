package command

import "github.com/spf13/cobra"

// newHello demonstrates a dependency-free one-shot command.
func newHello() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "一次性命令示例，不需要配置或数据库",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("hello from go-template")
			return nil
		},
	}
}
