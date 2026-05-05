package core

import (
	"fmt"
	"sort"
)

type Action struct {
	Name string
	Desc string
	Run  func(args []string) error
}

type Command struct {
	Name    string
	Desc    string
	Run     func(args []string) error
	Actions []*Action
}

var commands = make(map[string]*Command)

func RegisterCommand(c *Command) {
	if c == nil || c.Name == "" {
		panic("invalid command: must have Name")
	}
	if c.Run == nil && len(c.Actions) == 0 {
		panic("invalid command: must have Run or Actions")
	}
	if _, exists := commands[c.Name]; exists {
		panic("command already registered: " + c.Name)
	}
	commands[c.Name] = c
}

func Execute(name string, args []string) bool {
	cmd, ok := commands[name]
	if !ok {
		return false
	}

	if len(args) > 0 && len(cmd.Actions) > 0 {
		for _, a := range cmd.Actions {
			if a.Name == args[0] {
				if err := a.Run(args[1:]); err != nil {
					panic(err)
				}
				return true
			}
		}
	}

	if cmd.Run != nil {
		if err := cmd.Run(args); err != nil {
			panic(err)
		}
		return true
	}

	return false
}

func PrintHelp() {
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Println("Usage: <program> [-c config.yaml] <command> [action] [args]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, n := range names {
		cmd := commands[n]
		fmt.Printf("  %s\t%s\n", n, cmd.Desc)
		for _, a := range cmd.Actions {
			fmt.Printf("    %s\t%s\n", a.Name, a.Desc)
		}
	}
}
