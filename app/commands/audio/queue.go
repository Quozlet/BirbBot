package audio

import (
	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

// Queue lists the currently queued audio
type Queue struct{}

// ProcessMessage enqueues a List VoiceCommand
func (q Queue) ProcessMessage(
	response chan<- commands.MessageResponse,
	voiceCommandChannel chan<- VoiceCommand,
	m *discordgo.MessageCreate,
) (*Data, *commands.CommandError) {
	if !IsInVoiceChannel() {
		return nil, commands.NewError("Nothing in the queue because nothing is playing")
	}
	voiceCommandChannel <- List
	return nil, nil
}

// CommandList returns the list of aliases for the Queue Command
func (q Queue) CommandList() []string {
	return []string{"queue"}
}

// Help returns the help string for the queue command
func (q Queue) Help() string {
	return "`queue` lists the current queue"
}
