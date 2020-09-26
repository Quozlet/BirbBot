package audio

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"quozlet.net/birbbot/app/commands"
)

// Play is a Command to play audio from a file or URL.
type Play struct{}

// ProcessMessage will attempt to identify the kind of input, and play it.
// If no argument is provided, the stream is played if paused.
func (p Play) ProcessMessage(
	response chan<- commands.MessageResponse,
	voiceCommandChannel chan<- VoiceCommand,
	m *discordgo.MessageCreate,
) (*Data, *commands.CommandError) {
	splitContent := strings.Fields(m.Content)

	if len(splitContent[1:]) == 0 {
		if !IsInVoiceChannel() {
			return nil, commands.NewError("Nothing to play, not in voice")
		}
		response <- commands.MessageResponse{
			ChannelID: m.ChannelID,
			Reaction: commands.ReactionResponse{
				Add:       "▶️",
				MessageID: m.ID,
			},
		}
		voiceCommandChannel <- Start

		return nil, nil
	}

	url, err := url.Parse(splitContent[1])
	if err == nil {
		return playFromURL(url, response, m.ChannelID, splitContent[2:])
	}

	return nil, commands.CreateCommandError("Unrecognized format, can't enqueue to play", err)
}

// CommandList returns the list of aliases for the Play Command.
func (p Play) CommandList() []string {
	return []string{"play", "p"}
}

// Help returns the help string for the Play Command.
func (p Play) Help() string {
	return "`p`/`play` plays audio if it is paused\n" +
		"- `p`/`play` <audio URL> will enqueue that URL to be played\n" +
		"- `p`/`play` <audio URL> <title> will enqueue that URL with a title"
}

func playFromURL(
	url *url.URL,
	response chan<- commands.MessageResponse,
	channelID string,
	potentialTitle []string,
) (*Data, *commands.CommandError) {
	var commandError *commands.CommandError
	// Only valid URLs are enqueued.
	/* #nosec */
	resp, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url.String(), nil)
	if commandError = commands.CreateCommandError("Failed to load data", err); commandError != nil {
		return nil, commandError
	}

	encodeSession, err := dca.EncodeMem(resp.Body, dca.StdEncodeOptions)

	if commandError = commands.CreateCommandError("Failed to re-encode audio stream", err); commandError != nil {
		return nil, commandError
	}

	var title string
	if len(potentialTitle) != 0 {
		title = strings.Join(potentialTitle, " ")
	} else {
		title = strings.Title(
			strings.Join(
				strings.FieldsFunc(
					url.Path[strings.LastIndex(url.Path, "/")+1:],
					func(r rune) bool { return r == '-' },
				),
				" ",
			),
		)
	}
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message:   fmt.Sprintf("Queued \"%s\"", title),
	}
	log.Printf("Enqueueing %s", title)

	return &Data{
		audio: &audioSource{
			session: encodeSession,
		},
		mutex: &sync.Mutex{},
		Title: title,
	}, nil
}
