package commands

import (
	"os/exec"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

// FortuneCookie is a Command to return just a random fortune
type FortuneCookie struct{}

// Check asserts `fortune` is present as a command
func (fc FortuneCookie) Check(*pgxpool.Pool) error {
	_, err := exec.Command("fortune").Output()
	return err
}

// ProcessMessage returns a random fortune
func (fc FortuneCookie) ProcessMessage(*discordgo.MessageCreate, *pgxpool.Pool) (string, error) {
	fortune, err := exec.Command("fortune", "-a").Output()
	return string(fortune), err
}

// CommandList returns a list of aliases for the FortuneCookie Command
func (fc FortuneCookie) CommandList() []string {
	return []string{"!fortunecookie", "!fortune-cookie", "!fc"}
}

// Help returns the help message for the FortuneCookie Command
func (fc FortuneCookie) Help() string {
	return "Provides a random fortune"
}
