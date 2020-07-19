package commands

import (
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
)

var cows []string

// Cowsay is a command to have a cow say some text
type Cowsay struct{}

// Check asserts cowsay is present and a list of cows could be generated, returning an error if that isn't possible
func (c Cowsay) Check() error {
	cowsayOptions, err := exec.Command("cowsay", "-l").Output()
	if err != nil {
		return err
	}
	cows = strings.Fields(strings.Split(string(cowsayOptions), ":")[1])
	if len(cows) == 0 {
		return errors.New("Failed to generate a list of cows")
	}
	return nil
}

// ProcessMessage processes a message and returns a cow saying it, or an error if no message was supplied
func (c Cowsay) ProcessMessage(message ...string) (string, error) {
	if len(message) == 0 {
		return "", errors.New("Cows can't say anything unless you give them something to say, dingus")
	}
	// Could use `Choose` but re-using commands should only be used for generating output
	cow := cows[rand.Intn(len(cows))]
	cowMsg := strings.Join(message, " ")
	// OK to run user provided input
	/* #nosec */
	cowsay, err := exec.Command("cowsay", "-f", cow, cowMsg).Output()
	return fmt.Sprintf("```\n%s\n```", string(cowsay)), err
}

// CommandList returns a list of aliases for the Cowsay Command
func (c Cowsay) CommandList() []string {
	return []string{"!cowsay"}
}

// Help returns the help message for the Cowsay Command
func (c Cowsay) Help() string {
	return "Provides a random cow saying the provided message"
}

// Fortune is a Command to get a random cow saying a random fortune
type Fortune struct{}

// Check asserts `fortune` is present as a command
func (f Fortune) Check() error {
	_, err := exec.Command("fortune").Output()
	return err
}

// ProcessMessage returns a random cow saying a random message. The provided arguments are ignored
func (f Fortune) ProcessMessage(m ...string) (string, error) {
	fortune, err := exec.Command("fortune", "-a").Output()
	if err != nil {
		return "", err
	}
	return Cowsay{}.ProcessMessage(string(fortune))
}

// CommandList returns a list of aliases for the Fortune Command
func (f Fortune) CommandList() []string {
	return []string{"!fortune"}
}

// Help returns the help message for the Fortune Command
func (f Fortune) Help() string {
	return "Provides a random cow saying a random fortune"
}
