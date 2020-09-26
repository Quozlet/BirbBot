package animal

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"

	handler "quozlet.net/birbbot/util"

	"quozlet.net/birbbot/app/commands"
)

const (
	dogURL  = "https://shibe.online/api/shibes"
	birdURL = "https://shibe.online/api/birds"
	catURL  = "https://shibe.online/api/cats"
)

func fetchAnimal(ctx context.Context, url string) ([]string, *commands.CommandError) {
	var commandError *commands.CommandError

	log.Printf("Fetching animal from %s", url)
	// This can be statically proven to resolve to a const string
	/* #nosec */
	resp, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if commandError = commands.CreateCommandError(
		"Tried to get an image, but failed to initiate a connection to the API",
		err,
	); commandError != nil {
		return nil, commandError
	}

	body, err := ioutil.ReadAll(resp.Body)

	if commandError = commands.CreateCommandError(
		"Successfully made a connection, but an error occurred reading the response",
		err,
	); commandError != nil {
		return nil, commandError
	}
	defer handler.LogError(resp.Body.Close())

	chunkedBody := []rune(string(body))
	imageURL := string(chunkedBody[2 : len(chunkedBody)-2])

	return []string{imageURL}, nil
}
