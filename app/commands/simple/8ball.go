package simple

import (
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

var eightBallMessages = []string{
	"It is certain",
	"It is decidedly so",
	"Without a doubt",
	"Yes definitely",
	"You may rely on it",
	"As I see it, yes",
	"Most likely",
	"Outlook good", "Yes",
	"Signs point to yes",
	"Reply hazy try again",
	"Ask again later",
	"Better not tell you now",
	"Cannot predict now",
	"Concentrate and ask again",
	"Don't count on it",
	"My reply is no",
	"My sources say no",
	"Outlook not so good",
	"Very doubtful",
}

// EightBall is a Command to provide an answer to a yes/no question.
type EightBall struct{}

// Check always returns nil (all messages are guaranteed to be allocated).
func (e EightBall) Check() error {
	return nil
}

// ProcessMessage will return an error if no arguments are provided, otherwise a random message is chosen.
func (e EightBall) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
) *commands.CommandError {
	if len(strings.Fields(m.Content)) == 1 {
		return commands.NewError("You didn't give me anything to respond to")
	}
	// Cryptographically secure random numbers not necessary
	/* #nosec */
	response <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Message:   eightBallMessages[rand.Intn(len(eightBallMessages))],
	}

	return nil
}

// CommandList returns the invocable aliases for the 8 Ball Command.
func (e EightBall) CommandList() []string {
	return []string{"8", "8ball"}
}

// Help gives help information for the 8 Ball Command.
func (e EightBall) Help() string {
	return "Provides an answer to a yes/no question"
}
