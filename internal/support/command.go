package support

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

func Commands() map[string]*Command {
	return commands
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
