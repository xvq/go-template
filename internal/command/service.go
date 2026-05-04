package command

import (
	"github.com/user/go-template/internal/bootstrap"
	"github.com/user/go-template/internal/config"
	"github.com/user/go-template/internal/support"
)

func init() {
	support.RegisterCommand(&support.Command{
		Name: "server",
		Desc: "Start HTTP server",
		Run: func(args []string) error {
			bootstrap.StartServer(config.AppConfig)
			return nil
		},
	})

	support.RegisterCommand(&support.Command{
		Name: "worker",
		Desc: "Start cron worker",
		Run: func(args []string) error {
			bootstrap.StartWorker(config.AppConfig)
			return nil
		},
	})
}
