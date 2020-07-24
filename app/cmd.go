package app

import (
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
)

// Command is an interface that must be implemented for commands
type Command interface {
	// CommandList returns all aliases for the given command (must return at least one)
	CommandList() []string
	// Help returns the help message for the command
	Help() string
}

// SimpleCommand is a command that responds to a message with no other context
type SimpleCommand interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check() error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage(*discordgo.MessageCreate) (string, *commands.CommandError)
}

// PersistentCommand is a command that will persist some data into a database
type PersistentCommand interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check(*pgxpool.Pool) error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage(*discordgo.MessageCreate, *pgxpool.Pool) (string, *commands.CommandError)
}

// NoArgsCommand will always go through the same flow to response, irrespective of arguments
type NoArgsCommand interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check() error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage() (string, *commands.CommandError)
}
