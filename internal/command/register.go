package command

import "github.com/spf13/cobra"

// Register lists project commands. Configuration is read only at execution time.
func Register(path *string) []*cobra.Command {
	return []*cobra.Command{
		newAdminCreate(path),
		newQueueExample(path),
		newHello(),
	}
}
