package animal

import (
	"context"
	"net/url"

	"quozlet.net/birbbot/app/commands"
)

// Bird is a Command to get a random bird image.
type Bird struct{}

// Check if the bird URL is valid.
func (b Bird) Check() error {
	_, err := url.Parse(birdURL)

	return err
}

// ProcessMessage for a Bird Command (will return the URL for a random bird image).
func (b Bird) ProcessMessage() ([]string, *commands.CommandError) {
	return fetchAnimal(context.Background(), birdURL)
}

// CommandList returns applicable aliases for the Bird Command.
func (b Bird) CommandList() []string {
	return []string{"bird", "birb"}
}

// Help returns the Bird Command help message.
func (b Bird) Help() string {
	return "Provides a random image of a bird"
}
