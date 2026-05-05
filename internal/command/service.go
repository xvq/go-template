package command

import (
	"github.com/xvq/go-template/internal/bootstrap"
	"github.com/xvq/go-template/internal/core"
)

func init() {
	core.RegisterCommand(&core.Command{
		Name: "server",
		Desc: "Start HTTP server",
		Run: func(args []string) error {
			bootstrap.StartServer()
			return nil
		},
	})

	core.RegisterCommand(&core.Command{
		Name: "worker",
		Desc: "Start cron worker",
		Run: func(args []string) error {
			bootstrap.StartWorker()
			return nil
		},
	})
}
