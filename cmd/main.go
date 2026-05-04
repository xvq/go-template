package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	_ "github.com/xvq/go-template/internal/command"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/support"
)

func main() {
	var configPath string
	var showHelp bool
	flag.StringVar(&configPath, "c", "config.yaml", "config file path")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.Parse()

	tail := flag.Args()

	if showHelp || len(tail) == 0 || tail[0] == "help" {
		printHelp()
		return
	}

	config.AppConfig = config.Load(configPath)

	if support.Execute(tail[0], tail[1:]) {
		return
	}

	fmt.Fprintf(os.Stderr, "unknown command: %s\n", tail[0])
	printHelp()
	os.Exit(1)
}

func printHelp() {
	cmds := support.Commands()
	names := make([]string, 0, len(cmds))
	for n := range cmds {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Println("Usage: <program> [-c config.yaml] <command> [action] [args]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, n := range names {
		cmd := cmds[n]
		fmt.Printf("  %s\t%s\n", n, cmd.Desc)
		for _, a := range cmd.Actions {
			fmt.Printf("    %s\t%s\n", a.Name, a.Desc)
		}
	}
}
