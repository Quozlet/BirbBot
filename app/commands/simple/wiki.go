package simple

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	handler "quozlet.net/birbbot/util"

	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

const wikiURL = "https://en.wikipedia.org/api/rest_v1/page/summary/"

// Wiki is a Command to search for a Wikipedia article by title
type Wiki struct{}

// Check asserts the base Wikipedia API URL is valid
func (w Wiki) Check() error {
	_, err := url.Parse(wikiURL)
	return err
}

// ProcessMessage searches for a Wikipedia article by title
func (w Wiki) ProcessMessage(
	msgResponse chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
) *commands.CommandError {
	var commandError *commands.CommandError
	splitContent := strings.Fields(m.Content)
	if len(splitContent) == 1 {
		return commands.NewError("You didn't provide anything to look for on Wikipedia")
	}
	title := url.QueryEscape(strings.Join(splitContent[1:], "_"))
	wikiURL, err := url.Parse(wikiURL + title)
	if commandError = commands.CreateCommandError(
		"Failed to make that query into a request",
		err,
	); commandError != nil {
		return commandError
	}
	log.Println(wikiURL)
	response, err := http.Get(wikiURL.String())
	if commandError = commands.CreateCommandError(
		"Didn't hear back from Wikipedia about that article",
		err,
	); commandError != nil {
		return commandError
	}
	defer handler.LogError(response.Body.Close())
	wiki := wikiResponse{}
	if commandError = commands.CreateCommandError(
		"Heard back from Wikipedia, but couldn't process the response",
		json.NewDecoder(response.Body).Decode(&wiki),
	); commandError != nil {
		return commandError
	}
	if len(wiki.Description) == 0 || len(wiki.ContentURLs.Desktop.Page) == 0 {
		log.Printf("+%v", wiki)
		return commands.NewError("Didn't find a matching Wikipedia article")
	}
	msgResponse <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Message:   fmt.Sprintf("%s: %s", wiki.Description, wiki.ContentURLs.Desktop.Page),
	}
	return nil

}

// CommandList returns a list of aliases for the Wiki Command
func (w Wiki) CommandList() []string {
	return []string{"wiki"}
}

// Help returns the help message for the Wiki Command
func (w Wiki) Help() string {
	return "`wiki <title>` will make a best guess attempt to find the most relevant Wikipedia article"
}

type wikiResponse struct {
	Description string      `json:"description"`
	ContentURLs wikiContent `json:"content_urls"`
}

type wikiContent struct {
	Desktop wikiDesktopContent `json:"desktop"`
}

type wikiDesktopContent struct {
	Page string `json:"page"`
}
