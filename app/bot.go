package app

import (
	"log"
	"strings"

	"quozlet.net/birbbot/app/commands"
)

// Bot contains all commands and all alises for them (CommandList is equivalent to the keys to Commands)
type Bot struct {
	Commands    map[string]*Command
	CommandList []string
}

// Command is an interface that must be implemented for commands
type Command interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check() error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage(...string) (string, error)
	// CommandList returns all aliases for the given command (must return at least one)
	CommandList() []string
	// Help returns the help message for the command
	Help() string
}

// TODO: Automatically populate commands (requires some AST parser black magic)
// In the meantime newly added commands must implement all methods in the Command interface and be added to the list
func makeBot() *Bot {
	commandMap := make(map[string]*Command)
	for _, cmd := range []interface{}{
		commands.EightBall{},
		commands.Bird{},
		commands.Cat{},
		commands.Dog{},
		commands.Choose{},
		commands.Cowsay{},
		commands.Fortune{},
		commands.FortuneCookie{},
		commands.RSS{},
		commands.Weather{},
		commands.Forecast{},
	} {
		command, ok := cmd.(Command)
		if ok {
			err := command.Check()
			if err != nil {
				log.Println(err)
			} else {
				for _, alias := range command.CommandList() {
					if strings.HasPrefix(alias, "!") {
						commandMap[alias] = &command
					} else {
						log.Printf("Not registering %s (doesn't start with '!'", alias)
					}
				}
			}
		}
	}
	keys, i := make([]string, len(commandMap)), 0
	for key := range commandMap {
		keys[i] = key
		i++
	}
	log.Printf("Registered commands: %s", strings.Join(keys, ", "))
	return &Bot{Commands: commandMap, CommandList: keys}
}
