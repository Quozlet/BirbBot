package app

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
	"quozlet.net/birbbot/app/commands/audio"
	"quozlet.net/birbbot/app/commands/recurring"
)

// Prefix that all commands must begin with to be recognized.
const Prefix = '!'

// Command is an interface that must be implemented for commands.
type Command interface {
	// CommandList returns all aliases for the given command (must return at least one)
	CommandList() []string
	// Help returns the help message for the command
	Help() string
}

// SimpleCommand is a command that responds to a message with no other context.
type SimpleCommand interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check() error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage(chan<- commands.MessageResponse, *discordgo.MessageCreate) *commands.CommandError
}

// PersistentCommand is a command that will persist some data into a database.
type PersistentCommand interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check(*pgxpool.Pool) error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage(chan<- commands.MessageResponse, *discordgo.MessageCreate, *pgxpool.Pool) *commands.CommandError
}

// AudioCommand is a command that will return an Opus stream for a channel.
type AudioCommand interface {
	ProcessMessage(response chan<- commands.MessageResponse,
		voiceCommandChannel chan<- audio.VoiceCommand,
		m *discordgo.MessageCreate,
	) (*audio.Data, *commands.CommandError)
}

// NoArgsCommand will always go through the same flow to response, irrespective of arguments.
type NoArgsCommand interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check() error
	// ProcessMessage returns the response
	ProcessMessage() ([]string, *commands.CommandError)
}

// RecurringCommand will be run on a recurring basis, and return a map of channels to messages to post
// Note: It is not explicitly invoked, and some other command should handle populating data for it.
type RecurringCommand interface {
	// Check will check if there is any update. If an error occurs or there is no update, return nil
	Check(*pgxpool.Pool) map[string][]string
	// Frequency reports the preferred frequency for this command
	Frequency() recurring.Frequency
}

// BuildCommandName is a helper function to efficiently concatenate the current prefix with a command name.
func BuildCommandName(alias string) string {
	var builder strings.Builder

	builder.WriteRune(Prefix)
	builder.WriteString(alias)

	return builder.String()
}
