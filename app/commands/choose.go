package commands

import (
	"errors"
	"math/rand"
)

// Choose is a command to randomly select a choice from a set of (space delimited) options
type Choose struct{}

// Check returns nil since this requires nothing
func (c Choose) Check() error {
	return nil
}

// ProcessMessage processes a set of options to pick from, selecting one at random or returning an error if none are provided
func (c Choose) ProcessMessage(message ...string) (string, error) {
	if len(message) == 0 {
		return "", errors.New("Choices, choices. Do I choose the nothing you provided, or the nothing I'm going to provide in return? _Hint_: Give me something to pick smarty pants")
	}
	return message[rand.Intn(len(message))], nil
}

// CommandList returns the aliases for the Choose Command
func (c Choose) CommandList() []string {
	return []string{"!choose"}
}

// Help returns the help message for the Choose Command
func (c Choose) Help() string {
	return "Provides a random choice from one or more options"
}
