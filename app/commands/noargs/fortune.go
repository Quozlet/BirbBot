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

var (
	cows             []string
	errNoCows        = errors.New("failure occurred reading in the list of possible cows")
	errCowsayFailure = commands.NewError("So this is awkward but... I think the cow ate the fortune? " +
		"Something went wrong anyway")
)

// Fortune is a Command to get a random cow saying a random fortune.
type Fortune struct{}

// Check asserts `fortune` is present as a command.
func (f Fortune) Check() error {
	var err error
	cows, err = simple.PopulateCows()

	if err != nil {
		return err
	}

	if len(cows) == 0 {
		return errNoCows
	}

	if _, fortuneErr := exec.LookPath("fortune"); fortuneErr != nil {
		log.Println(fortuneErr)

		return errMissingFortuneProgram
	}

	return nil
}

// ProcessMessage returns a random cow saying a random message. The provided arguments are ignored.
func (f Fortune) ProcessMessage() ([]string, *commands.CommandError) {
	fortune, err := exec.Command("fortune", "-a").Output()
	if err != nil {
		log.Println(err)

		return nil, errNoFortune
	}
	// OK to run user provided input
	/* #nosec */
	cowsay, cowsayErr := exec.Command("cowsay", "-f", cows[rand.Intn(len(cows))], string(fortune)).Output()
	if cowsayErr != nil {
		log.Println(cowsayErr)

		return nil, errCowsayFailure
	}

	return []string{fmt.Sprintf("```\n%s\n```", string(cowsay))}, nil
}

// CommandList returns a list of aliases for the Fortune Command.
func (f Fortune) CommandList() []string {
	return []string{"fortune"}
}

// Help returns the help message for the Fortune Command.
func (f Fortune) Help() string {
	return "Provides a random cow saying a random fortune"
}
