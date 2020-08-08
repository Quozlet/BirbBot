package noargs

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os/exec"

	"quozlet.net/birbbot/app/commands"
	"quozlet.net/birbbot/app/commands/simple"
)

var cows []string

// Fortune is a Command to get a random cow saying a random fortune
type Fortune struct{}

// Check asserts `fortune` is present as a command
func (f Fortune) Check() error {
	var err error
	cows, err = simple.PopulateCows()
	if err != nil {
		return err
	}
	if len(cows) == 0 {
		return errors.New("Failure occurred reading in the list of possible cows")
	}
	_, fortuneErr := exec.LookPath("fortune")
	if fortuneErr != nil {
		log.Println(fortuneErr)
		return errors.New("'fortune' is not installed")
	}
	return nil
}

// ProcessMessage returns a random cow saying a random message. The provided arguments are ignored
func (f Fortune) ProcessMessage() ([]string, *commands.CommandError) {
	fortune, err := exec.Command("fortune", "-a").Output()
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Doubt is not a pleasant condition, but... just kidding, I didn't get a fortune. " +
			" Guess the fortune teller fell asleep ¯\\_(ツ)_/¯")
	}
	// OK to run user provided input
	/* #nosec */
	cowsay, cowsayErr := exec.Command("cowsay", "-f", cows[rand.Intn(len(cows))], string(fortune)).Output()
	if cowsayErr != nil {
		log.Println(cowsayErr)
		return nil, commands.NewError("So this is awkward but... I think the cow ate the fortune? " +
			"Something went wrong anyway")
	}
	return []string{fmt.Sprintf("```\n%s\n```", string(cowsay))}, nil
}

// CommandList returns a list of aliases for the Fortune Command
func (f Fortune) CommandList() []string {
	return []string{"!fortune"}
}

// Help returns the help message for the Fortune Command
func (f Fortune) Help() string {
	return "Provides a random cow saying a random fortune"
}
