package noargs

import (
	"quozlet.net/birbbot/app/commands"
)

// Source is a Command to provide a link to the source code for this bot.
type Source struct{}

// Check always returns nil.
func (s Source) Check() error {
	return nil
}

// ProcessMessage returns the link to the source code of this bot.
func (s Source) ProcessMessage() ([]string, *commands.CommandError) {
	return []string{"https://github.com/Quozlet/BirbBot"}, nil
}

// CommandList returns the invocable aliases for the Source Command.
func (s Source) CommandList() []string {
	return []string{"source"}
}

// Help gives help information for the Source Command.
func (s Source) Help() string {
	return "Get the link to this bot's source code"
}
