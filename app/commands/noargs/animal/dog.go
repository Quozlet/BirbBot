package animal

import (
	"net/url"

	"quozlet.net/birbbot/app/commands"
)

// Dog is a Command to get a random dog image
type Dog struct{}

// Check if the dog URL is valid
func (d Dog) Check() error {
	_, err := url.Parse(dogURL)
	return err
}

// ProcessMessage for a Dog Command (will return the URL for a random dog (specifically shibe) image)
func (d Dog) ProcessMessage() ([]string, *commands.CommandError) {
	return fetchAnimal(dogURL)
}

// CommandList returns applicable aliases for Dog Command
func (d Dog) CommandList() []string {
	return []string{"dog", "shibe"}
}

// Help returns the Dog Command help message
func (d Dog) Help() string {
	return "Provides a random image of a dog/shibe"
}
