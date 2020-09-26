package animal

import (
	"context"
	"net/url"

	"quozlet.net/birbbot/app/commands"
)

// Cat is a Command to get a random cat image.
type Cat struct{}

// Check if the cat URL is valid.
func (c Cat) Check() error {
	_, err := url.Parse(catURL)

	return err
}

// ProcessMessage for a Cat Command (will return the URL for a random cat image).
func (c Cat) ProcessMessage() ([]string, *commands.CommandError) {
	return fetchAnimal(context.Background(), catURL)
}

// CommandList returns applicable aliases for Cat Command.
func (c Cat) CommandList() []string {
	return []string{"cat"}
}

// Help returns the Cat Command help message.
func (c Cat) Help() string {
	return "Provides a random image of a cat"
}
