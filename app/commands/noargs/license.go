package noargs

import (
	"quozlet.net/birbbot/app/commands"
)

// License is a Command to provide a link to the license for this bot's source code
type License struct{}

// Check always returns nil
func (l License) Check() error {
	return nil
}

// ProcessMessage returns the link to the license for the source code of this bot
func (l License) ProcessMessage() ([]string, *commands.CommandError) {
	return []string{"This bot's source code is licensed under the The Open Software License 3.0 (https://spdx.org/licenses/OSL-3.0.html)"}, nil
}

// CommandList returns the invocable aliases for the License Command
func (l License) CommandList() []string {
	return []string{"license"}
}

// Help gives help information for the License Command
func (l License) Help() string {
	return "Provides the software license that applies to this bot's source code"
}
