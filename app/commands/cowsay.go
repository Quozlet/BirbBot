package commands

import (
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

var cows []string

// Cowsay is a command to have a cow say some text
type Cowsay struct{}

// Check asserts cowsay is present and a list of cows could be generated, returning an error if that isn't possible
func (c Cowsay) Check(*pgxpool.Pool) error {
	cowsayOptions, err := exec.Command("cowsay", "-l").Output()
	if err != nil {
		log.Println(err)
		return &CommandError{msg: fmt.Sprintf("%s failed check, 'cowsay' failed to run", strings.Join(c.CommandList(), ","))}
	}
	cows = strings.Fields(strings.Split(string(cowsayOptions), ":")[1])
	if len(cows) == 0 {
		return &CommandError{msg: "Failed to generate a list of cows"}
	}
	return nil
}

// ProcessMessage processes a message and returns a cow saying it, or an error if no message was supplied
func (c Cowsay) ProcessMessage(m *discordgo.MessageCreate, _ *pgxpool.Pool) (string, error) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) == 1 {
		return "", &CommandError{msg: "Cows can't say anything unless you give them something to say, dingus"}
	}
	cow := cows[rand.Intn(len(cows))]
	cowMsg := string([]rune(m.Content)[len(splitContent[0])+1:])
	// OK to run user provided input
	/* #nosec */
	cowsay, err := exec.Command("cowsay", "-f", cow, cowMsg).Output()
	if err != nil {
		log.Println(err)
		return "", &CommandError{msg: "Something bad happened when I asked the cow to say that..."}
	}
	return fmt.Sprintf("```\n%s\n```", string(cowsay)), nil
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
func (f Fortune) Check(*pgxpool.Pool) error {
	_, err := exec.Command("fortune").Output()
	if err != nil {
		log.Println(err)
		return &CommandError{msg: fmt.Sprintf("%s failed check, 'fortune' failed to run", strings.Join(f.CommandList(), ","))}
	}
	return nil
}

// ProcessMessage returns a random cow saying a random message. The provided arguments are ignored
func (f Fortune) ProcessMessage(m *discordgo.MessageCreate, _ *pgxpool.Pool) (string, error) {
	fortune, err := exec.Command("fortune", "-a").Output()
	if err != nil {
		log.Println(err)
		return "", &CommandError{msg: "Doubt is not a pleasant condition, but... just kidding, I didn't get a fortune." +
			" Guess the fortune teller fell asleep ¯\\_(ツ)_/¯"}
	}
	cow := cows[rand.Intn(len(cows))]
	// OK to run user provided input
	/* #nosec */
	cowsay, err := exec.Command("cowsay", "-f", cow, string(fortune)).Output()
	if err != nil {
		log.Println(err)
		return "", &CommandError{msg: "So this is awkward but... I think the cow ate the fortune?" +
			" Something went wrong anyway"}
	}
	return fmt.Sprintf("```\n%s\n```", string(cowsay)), nil
}

// CommandList returns a list of aliases for the Fortune Command
func (f Fortune) CommandList() []string {
	return []string{"!fortune"}
}

// Help returns the help message for the Fortune Command
func (f Fortune) Help() string {
	return "Provides a random cow saying a random fortune"
}
