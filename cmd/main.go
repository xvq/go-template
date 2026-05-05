package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/xvq/go-template/internal/command"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/core"
)

func main() {
	var configPath string
	var showHelp bool
	flag.StringVar(&configPath, "c", "config.yaml", "config file path")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.Parse()

	tail := flag.Args()

	if showHelp || len(tail) == 0 || tail[0] == "help" {
		core.PrintHelp()
		return
	}

	config.Load(configPath)

	if core.Execute(tail[0], tail[1:]) {
		return
	}

	fmt.Fprintf(os.Stderr, "unknown command: %s\n", tail[0])
	core.PrintHelp()
	os.Exit(1)
}
