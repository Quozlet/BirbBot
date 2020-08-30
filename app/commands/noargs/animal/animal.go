package animal

import (
	"io/ioutil"
	"log"
	"net/http"

	handler "quozlet.net/birbbot/util"

	"quozlet.net/birbbot/app/commands"
)

const dogURL = "https://shibe.online/api/shibes"
const birdURL = "https://shibe.online/api/birds"
const catURL = "https://shibe.online/api/cats"

func fetchAnimal(url string) ([]string, *commands.CommandError) {
	var commandError *commands.CommandError
	log.Printf("Fetching animal from %s", url)
	// This can be statically proven to resolve to a const string
	/* #nosec */
	resp, err := http.Get(url)
	if commandError = commands.CreateCommandError(
		"Tried to get an image, but failed to initiate a connection to the API",
		err,
	); commandError != nil {
		return nil, commandError
	}
	body, err := ioutil.ReadAll(resp.Body)
	if commandError = commands.CreateCommandError(
		"Successfully made a connection, but an error occured reading the response",
		err,
	); commandError != nil {
		return nil, commandError
	}
	defer handler.LogError(resp.Body.Close())
	chunkedBody := []rune(string(body))
	imageURL := string(chunkedBody[2 : len(chunkedBody)-2])
	return []string{imageURL}, nil
}
