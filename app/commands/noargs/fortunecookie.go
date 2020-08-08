package noargs

import (
	"errors"
	"log"
	"os/exec"

	"quozlet.net/birbbot/app/commands"
)

// FortuneCookie is a Command to return just a random fortune
type FortuneCookie struct{}

// Check asserts `fortune` is present as a command
func (fc FortuneCookie) Check() error {
	_, err := exec.LookPath("fortune")
	if err != nil {
		log.Println(err)
		return errors.New("'fortune' is not installed")
	}
	return nil
}

// ProcessMessage returns a random fortune
func (fc FortuneCookie) ProcessMessage() ([]string, *commands.CommandError) {
	fortune, err := exec.Command("fortune", "-a").Output()
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Doubt is not a pleasant condition, but... just kidding, I didn't get a fortune. " +
			"Guess the fortune teller fell asleep ¯\\_(ツ)_/¯")
	}
	return []string{string(fortune)}, nil
}

// CommandList returns a list of aliases for the FortuneCookie Command
func (fc FortuneCookie) CommandList() []string {
	return []string{"!fortunecookie", "!fortune-cookie", "!fc"}
}

// Help returns the help message for the FortuneCookie Command
func (fc FortuneCookie) Help() string {
	return "Provides a random fortune"
}
