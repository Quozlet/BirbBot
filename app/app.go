package app

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands"
)

// Command is an interface that must be implemented for commands
type Command interface {
	// Check asserts all preconditions are met, and returns an error if they are not
	Check() error
	// ProcessMessage processes all additional arguments to the command (split on whitespace)
	ProcessMessage(...string) (string, error)
	// CommandList returns all aliases for the given command (must return at least one)
	CommandList() []string
	// Help returns the help message for the command
	Help() string
}

// Start a Discord session for a given token
func Start(secret string) (*discordgo.Session, error) {
	if len(secret) == 0 {
		return nil, errors.New("Not attempting connection, secret seems incorrect")
	}
	commandMap, commandList := discoverCommand()
	session, sessionError := discordgo.New("Bot " + secret)
	if sessionError != nil {
		log.Println("Unable to create Discord session")
		return nil, sessionError
	}
	log.Println("Successfully created Discord session")
	// TODO: If panicking while processing a command, error instead of crashing
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages without the '!' prefix or with own ID
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!") {
			return
		}
		commandHandler(s, m, commandMap, commandList)
	})
	sessionError = session.Open()
	if sessionError != nil {
		log.Println("Failed to open WebSocket connection to Discord servers")
		return nil, sessionError
	}
	log.Println("Opened WebSocket connection to Discord")
	return session, nil
}

func commandHandler(s *discordgo.Session, m *discordgo.MessageCreate, commandMap map[string]*Command, commandList []string) {
	content := strings.Fields(strings.ToLower(m.Content))
	cmd := commandMap[content[0]]
	log.Printf("Ack: %s", m.Content)
	_, discordMsgErr := s.ChannelMessageSend(m.ChannelID, func() string {
		if cmd != nil {
			reactionErr := s.MessageReactionAdd(m.ChannelID, m.Message.ID, "✅")
			if reactionErr != nil {
				log.Println(reactionErr)
			}
			defer func() {
				reactionErr := s.MessageReactionRemove(m.ChannelID, m.Message.ID, "✅", s.State.User.ID)
				if reactionErr != nil {
					log.Println(reactionErr)
				}
			}()
			response, err := (*cmd).ProcessMessage(content[1:]...)
			if err != nil {
				log.Printf("An error occurred processing %s: %s", content, err.Error())
				reactionErr := s.MessageReactionRemove(m.ChannelID, m.Message.ID, "✅", s.State.User.ID)
				if reactionErr != nil {
					log.Println(err)
				}
				failureReactionErr := s.MessageReactionAdd(m.ChannelID, m.Message.ID, "❗")
				if failureReactionErr != nil {
					log.Println(failureReactionErr)
				}
				return err.Error()
			}
			log.Printf("Responded ok to %s", m.Content)
			return response

		}
		// Handle '!help', '!license', '!source'
		switch content[0] {
		case "!help":
			if len(content[1:]) == 0 || commandMap["!"+content[1]] == nil {
				return fmt.Sprintf("Available commands:\n`%s`, `!license` (the software license that applies to this bot's source code), `!source` (a link to this bot's source code)\n\n(For more information type asking for `!help <command name>`)", strings.Join(commandList, "`, `"))
			}
			return (*commandMap["!"+content[1]]).Help()

		case "!license":
			return "https://spdx.org/licenses/OSL-3.0.html"

		case "!source":
			return "https://github.com/Quozlet/BirbBot"

		default:
			log.Printf("Unrecognized command: %s", m.Content)
			return fmt.Sprintf("Unrecognized command: `%s`", content[0])
		}

	}())
	if discordMsgErr != nil {
		log.Printf("Failed to respond: %s", discordMsgErr)
	}

}

// TODO: Automatically populate commands (requires some AST parser black magic)
// In the meantime newly added commands must implement all methods in the Command interface and be added to the list
func discoverCommand() (map[string]*Command, []string) {
	commandMap := make(map[string]*Command)
	for _, cmd := range []interface{}{
		commands.EightBall{},
		commands.Bird{},
		commands.Cat{},
		commands.Dog{},
		commands.Choose{},
		commands.Cowsay{},
		commands.Fortune{},
		commands.FortuneCookie{},
		commands.RSS{},
		commands.Weather{},
		commands.Forecast{},
	} {
		command, ok := cmd.(Command)
		if ok {
			err := command.Check()
			if err != nil {
				log.Println(err)
			} else {
				for _, alias := range command.CommandList() {
					if strings.HasPrefix(alias, "!") {
						commandMap[alias] = &command
					} else {
						log.Printf("Not registering %s (doesn't start with '!')", alias)
					}
				}
			}
		}
	}
	keys, i := make([]string, len(commandMap)), 0
	for key := range commandMap {
		keys[i] = key
		i++
	}
	log.Printf("Registered commands: %s", strings.Join(keys, ", "))
	return commandMap, keys
}
