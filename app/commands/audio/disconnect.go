package audio

import (
	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

// Disconnect clears the queue and leaves the voice channel.
type Disconnect struct{}

// ProcessMessage enqueues a Disconnect VoiceCommand.
func (d Disconnect) ProcessMessage(
	response chan<- commands.MessageResponse,
	voiceCommandChannel chan<- VoiceCommand,
	m *discordgo.MessageCreate,
) (*Data, *commands.CommandError) {
	if !IsInVoiceChannel() {
		return nil, commands.NewError("Not disconnecting, no audio is playing")
	}
	response <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Reaction: commands.ReactionResponse{
			Add:       "🔌",
			MessageID: m.ID,
		},
	}
	voiceCommandChannel <- Leave

	return nil, nil
}

// CommandList returns the list of aliases for the Disconnect Command.
func (d Disconnect) CommandList() []string {
	return []string{"disconnect", "dc"}
}

// Help returns the help string for the Disconnect Command.
func (d Disconnect) Help() string {
	return "`disconnect`/`dc` enqueues a disconnect, which will run after the currently playing track, if there is any"
}
