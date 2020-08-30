package animal

import (
	"io/ioutil"
	"log"
	"net/http"

	"quozlet.net/birbbot/app/commands"
)

const dogURL = "https://shibe.online/api/shibes"
const birdURL = "https://shibe.online/api/birds"
const catURL = "https://shibe.online/api/cats"

func fetchAnimal(url string) ([]string, *commands.CommandError) {
	log.Printf("Fetching animal from %s", url)
	// This can be statically proven to resolve to a const string
	/* #nosec */
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Tried to get an image, but failed to initiate a connection to the API")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Successfully made a connection, but an error occured reading the response")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err)
		}
	}()
	chunkedBody := []rune(string(body))
	imageURL := string(chunkedBody[2 : len(chunkedBody)-2])
	return []string{imageURL}, nil
}
