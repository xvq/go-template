package cli

import (
	"runtime"

	"github.com/spf13/cobra"
)

// Set by the build script through -ldflags -X; immutable during execution.
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func newVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "查看程序版本与构建信息",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("Version:    %s\nCommit:     %s\nBuild time: %s\nGo version: %s\n", version, commit, buildTime, runtime.Version())
			return nil
		},
	}
}
