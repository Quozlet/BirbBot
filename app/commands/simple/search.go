package simple

import (
	"context"
	"encoding/json"
	"errors"
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

var errNoSearchResults = errors.New("no results found")

// Search is a command to make a websearch.
type Search struct{}

// Check that the query URL is valid.
func (s Search) Check() error {
	_, err := url.Parse(searchURL)

	return err
}

// ProcessMessage with search query and return first result.
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

	search, err := url.Parse(searchURL)

	handler.LogError(err)

	if commandError = commands.CreateCommandError(
		"Failed to make that query into searchable text",
		err,
	); commandError != nil {
		return commandError
	}

	q := search.Query()
	q.Set("q", url.QueryEscape(strings.Join(splitContent[1:], " ")))
	search.RawQuery = q.Encode()

	result, err := fetchSearchResults(search)
	if commandError = commands.CreateCommandError("Failed to fetch search results", err); commandError != nil {
		return commandError
	}

	msgResponse <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Message: fmt.Sprintf("**%s**\n%s\n%s",
			result.Results[0].Title,
			result.Results[0].Content,
			result.Results[0].URL),
	}

	return nil
}

// CommandList returns a list of aliases for the Search Command.
func (s Search) CommandList() []string {
	return []string{"s", "search"}
}

// Help returns the help message for the Weather Command.
func (s Search) Help() string {
	return "`s`/`search` to perform a web search\n" +
		"_Powered by searX_"
}

// SearchResponse contains all results.
type SearchResponse struct {
	Results []Result `json:"results"`
}

// Result is a single result for a given query.
type Result struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
}

func fetchSearchResults(search *url.URL) (*SearchResponse, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, search.String(), nil)

	if err := commands.CreateCommandError(
		"Failure occurred while constructing request",
		err,
	); err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "birbbot")
	log.Printf("Searching %s", search.String())

	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return nil, err
	}
	defer handler.LogError(response.Body.Close())

	resp := SearchResponse{}

	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, err
	}

	if len(resp.Results) == 0 {
		return nil, errNoSearchResults
	}

	return &resp, nil
}
