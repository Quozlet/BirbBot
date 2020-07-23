package commands

// CommandError contains user-facing information about an error that occurred processing a command
type CommandError struct {
	msg string
}

func (cErr *CommandError) Error() string {
	return cErr.msg
}
