package commands

import "os/exec"

// Fortune is a Command to get a random cow saying a random fortune
type Fortune struct{}

// Check asserts `fortune` is present as a command
func (f Fortune) Check() error {
	_, err := exec.Command("fortune").Output()
	return err
}

// ProcessMessage returns a random cow saying a random message. The provided arguments are ignored
func (f Fortune) ProcessMessage(m ...string) (string, error) {
	fortune, err := FortuneCookie{}.ProcessMessage(m...)
	if err != nil {
		return "", err
	}
	return Cowsay{}.ProcessMessage(fortune)
}

// CommandList returns a list of aliases for the Fortune Command
func (f Fortune) CommandList() []string {
	return []string{"!fortune"}
}

// Help returns the help message for the Fortune Command
func (f Fortune) Help() string {
	return "Provides a random cow saying a random fortune"
}

// FortuneCookie is a Command to return just a random fortune
type FortuneCookie struct{}

// Check asserts `fortune` is present as a command
func (fc FortuneCookie) Check() error {
	_, err := exec.Command("fortune").Output()
	return err
}

// ProcessMessage returns a random fortune
func (fc FortuneCookie) ProcessMessage(...string) (string, error) {
	fortune, err := exec.Command("ortune", "-a").Output()
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
