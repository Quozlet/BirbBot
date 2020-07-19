package commands

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

const dogURL = "https://shibe.online/api/shibes"
const birdURL = "https://shibe.online/api/birds"
const catURL = "https://shibe.online/api/cats"

// Dog is a Command to get a random dog image
type Dog struct{}

// Check if the dog URL is valid
func (d Dog) Check() error {
	_, err := url.Parse(dogURL)
	return err
}

// ProcessMessage for a Dog Command (will return the URL for a random dog (specifically shibe) image)
func (d Dog) ProcessMessage(...string) (string, error) {
	return fetchAnimal("https://shibe.online/api/shibes")
}

// CommandList returns applicable aliases for Dog Command
func (d Dog) CommandList() []string {
	return []string{"!dog", "!shibe"}
}

// Help returns the Dog Command help message
func (d Dog) Help() string {
	return "Provides a random image of a dog/shibe"
}

// Cat is a Command to get a random cat image
type Cat struct{}

// Check if the cat URL is valid
func (c Cat) Check() error {
	_, err := url.Parse("https://shibe.online/api/cats")
	return err
}

// ProcessMessage for a Cat Command (will return the URL for a random cat image)
func (c Cat) ProcessMessage(...string) (string, error) {
	return fetchAnimal("https://shibe.online/api/cats")
}

// CommandList returns applicable aliases for Cat Command
func (c Cat) CommandList() []string {
	return []string{"!cat"}
}

// Help returns the Cat Command help message
func (c Cat) Help() string {
	return "Provides a random image of a cat"
}

// Bird is a Command to get a random bird image
type Bird struct{}

// Check if the bird URL is valid
func (b Bird) Check() error {
	_, err := url.Parse("https://shibe.online/api/birds")
	return err
}

// ProcessMessage for a Bird Command (will return the URL for a random bird image)
func (b Bird) ProcessMessage(...string) (string, error) {
	return fetchAnimal("https://shibe.online/api/birds")
}

// CommandList returns applicable alises for the Bird Command
func (b Bird) CommandList() []string {
	return []string{"!bird", "!birb"}
}

// Help returns the Bird Command hepl message
func (b Bird) Help() string {
	return "Provides a random image of a bird"
}

func fetchAnimal(url string) (string, error) {
	log.Printf("Fetching animal from %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	chunkedBody := []rune(string(body))
	imageURL := string(chunkedBody[2 : len(chunkedBody)-2])
	return imageURL, nil
}
