package audio

import (
	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

// Pause pauses the currently playing stream.
type Pause struct{}

// ProcessMessage will pause the stream (if it isn't already).
func (p Pause) ProcessMessage(
	response chan<- commands.MessageResponse,
	voiceCommandChannel chan<- VoiceCommand,
	m *discordgo.MessageCreate,
) (*Data, *commands.CommandError) {
	if !IsInVoiceChannel() {
		return nil, commands.NewError("Cannot pause, nothing is playing")
	}
	response <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Reaction: commands.ReactionResponse{
			Add:       "⏸️",
			MessageID: m.ID,
		},
	}
	voiceCommandChannel <- Stop

	return nil, nil
}

// CommandList returns the list of aliases for the Pause Command.
func (p Pause) CommandList() []string {
	return []string{"pause", "stop"}
}

// Help returns the help string for the Pause Command.
func (p Pause) Help() string {
	return "`pause`/`stop` pauses the currently playing audio"
}
