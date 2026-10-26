package cli

import (
	"github.com/spf13/cobra"
	template "github.com/xvq/go-template"
	"os"
)

func newConfigInit(path *string) *cobra.Command {
	return &cobra.Command{Use: "config:init", Short: "生成配置文件（不覆盖已有文件）", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.OpenFile(*path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = f.Write(template.ExampleConfig)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		cmd.Printf("配置已生成: %s\n", *path)
		return nil
	}}
}
