package simple

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

var cows []string

// Cowsay is a command to have a cow say some text
type Cowsay struct{}

// Check asserts cowsay is present and a list of cows could be generated, returning an error if that isn't possible
func (c Cowsay) Check() error {
	var err error
	cows, err = PopulateCows()
	if err != nil {
		return err
	}
	if len(cows) == 0 {
		return errors.New("Failure occurred reading in the list of possible cows")
	}
	return nil
}

// ProcessMessage processes a message and returns a cow saying it, or an error if no message was supplied
func (c Cowsay) ProcessMessage(m *discordgo.MessageCreate) ([]string, *commands.CommandError) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) == 1 {
		return nil, commands.NewError("Cows can't say anything unless you give them something to say, dingus")
	}
	cow := cows[rand.Intn(len(cows))]
	cowMsg := string([]rune(m.Content)[len(splitContent[0])+1:])
	// OK to run user provided input
	/* #nosec */
	cowsay, err := exec.Command("cowsay", "-f", cow, cowMsg).Output()
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Something bad happened when I asked the cow to say that...")
	}
	return []string{fmt.Sprintf("```\n%s\n```", string(cowsay))}, nil
}

// CommandList returns a list of aliases for the Cowsay Command
func (c Cowsay) CommandList() []string {
	return []string{"!cowsay"}
}

// Help returns the help message for the Cowsay Command
func (c Cowsay) Help() string {
	return "Provides a random cow saying the provided message"
}

// PopulateCows asserts cowsay is present and provides a list of all possible cows
func PopulateCows() ([]string, error) {
	if len(cows) != 0 {
		return cows, nil
	}
	_, err := exec.LookPath("cowsay")
	if err != nil {
		log.Println(err)
		return nil, errors.New("'cowsay' is not installed")
	}
	cowsayOptions, err := exec.Command("cowsay", "-l").Output()
	if err != nil {
		log.Println(err)
		return nil, errors.New("'cowsay' is installed, but failed to give a list of cows")
	}
	return strings.Fields(strings.Split(string(cowsayOptions), ":")[1]), nil
}
