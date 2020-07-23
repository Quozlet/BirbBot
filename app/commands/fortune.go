package commands

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

// FortuneCookie is a Command to return just a random fortune
type FortuneCookie struct{}

// Check asserts `fortune` is present as a command
func (fc FortuneCookie) Check(*pgxpool.Pool) error {
	_, err := exec.Command("fortune").Output()
	if err != nil {
		log.Println(err)
		return &CommandError{msg: fmt.Sprintf("%s failed check, 'fortune' failed to run", strings.Join(fc.CommandList(), ","))}
	}
	return nil
}

// ProcessMessage returns a random fortune
func (fc FortuneCookie) ProcessMessage(*discordgo.MessageCreate, *pgxpool.Pool) (string, error) {
	fortune, err := exec.Command("fortune", "-a").Output()
	if err != nil {
		log.Println(err)
		return "", &CommandError{msg: "Doubt is not a pleasant condition, but... just kidding, I didn't get a fortune." +
			" Guess the fortune teller fell asleep ¯\\_(ツ)_/¯"}
	}
	return string(fortune), nil
}

// CommandList returns a list of aliases for the FortuneCookie Command
func (fc FortuneCookie) CommandList() []string {
	return []string{"!fortunecookie", "!fortune-cookie", "!fc"}
}

// Help returns the help message for the FortuneCookie Command
func (fc FortuneCookie) Help() string {
	return "Provides a random fortune"
}
