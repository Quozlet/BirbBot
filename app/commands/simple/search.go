package simple

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
	handler "quozlet.net/birbbot/util"
)

const searchURL = "https://searx.xyz/search?format=json&lang=en"

// Search is a command to make a websearch
type Search struct{}

// Check that the query URL is valid
func (s Search) Check() error {
	_, err := url.Parse(searchURL)
	return err
}

// ProcessMessage with search query and return first result
func (s Search) ProcessMessage(
	msgResponse chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
) *commands.CommandError {
	var commandError *commands.CommandError
	splitContent := strings.Fields(m.Content)
	if len(splitContent) == 1 {
		return commands.NewError("Can't search for nothing." +
			" I mean, I can search for 'nothing', but you gave me nothing to search..." +
			" Listen, you get it." +
			" Provide some input next time")
	}
	searchURL, err := url.Parse(searchURL)
	if commandError = commands.CreateCommandError(
		"Failed to make that query into searchable text",
		err,
	); commandError != nil {
		return commandError
	}
	q := searchURL.Query()
	q.Set("q", url.QueryEscape(strings.Join(splitContent[1:], " ")))
	searchURL.RawQuery = q.Encode()
	request, err := http.NewRequest("GET", searchURL.String(), nil)
	if commandError = commands.CreateCommandError(
		"Failure occurred while constructing request",
		err,
	); commandError != nil {
		return commandError
	}
	request.Header.Set("User-Agent", "birbbot")
	log.Printf("Searching %s", searchURL.String())
	response, err := (&http.Client{}).Do(request)
	if commandError = commands.CreateCommandError(
		"Failed to hear back from the server",
		err,
	); commandError != nil {
		return commandError
	}
	defer handler.LogError(response.Body.Close())
	search := SearchResponse{}
	if commandError = commands.CreateCommandError(
		"Heard back, but couldn't process the response",
		json.NewDecoder(response.Body).Decode(&search),
	); commandError != nil {
		return commandError
	}
	if len(search.Results) == 0 {
		return commands.NewError("No results found")
	}
	msgResponse <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Message: fmt.Sprintf("**%s**\n%s\n%s",
			search.Results[0].Title,
			search.Results[0].Content,
			search.Results[0].URL),
	}
	return nil
}

// CommandList returns a list of aliases for the Search Command
func (s Search) CommandList() []string {
	return []string{"s", "search"}
}

// Help returns the help message for the Weather Command
func (s Search) Help() string {
	return "`s`/`search` to perform a web search\n" +
		"_Powered by searX_"
}

// SearchResponse contains all results
type SearchResponse struct {
	Results []Result `json:"results"`
}

// Result is a single result for a given query
type Result struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
}
