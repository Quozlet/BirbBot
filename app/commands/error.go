package commands

import (
	"log"
)

// CommandError contains user-facing information about an error that occurred processing a command
type CommandError struct {
	msg string
}

func (cErr *CommandError) Error() string {
	return cErr.msg
}

// NewError creation for a Command
func NewError(msg string) *CommandError {
	return &CommandError{msg: msg}
}

// CreateCommandError takes in a message and error
// If the error is not nill, a new CommandError is returned with the provided message
func CreateCommandError(msg string, err error) *CommandError {
	if err != nil {
		log.Println(err)
		return NewError(msg)
	}
	return nil
}
