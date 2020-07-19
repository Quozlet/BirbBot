package app

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var bot *Bot

// Start a Discord session for a given token
func Start(secret string) (*discordgo.Session, error) {
	if len(secret) == 0 {
		return nil, errors.New("Not attempting connection, secret seems incorrect")
	}
	bot = makeBot()
	session, err := discordgo.New("Bot " + secret)
	if err != nil {
		log.Println("Unable to create Discord session")
		return nil, err
	}
	log.Println("Successfully created Discord session")
	// TODO: If panicking while processing a command, error instead of crashing
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages without the '!' prefix or with own ID
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!") {
			return
		}
		content := strings.Fields(strings.ToLower(m.Content))
		cmd := bot.Commands[content[0]]
		// If command exists
		if cmd != nil {
			log.Printf("Preparing to respond to %s", m.Content)
			response, err := (*cmd).ProcessMessage(content[1:]...)
			if err != nil {
				log.Printf("An error occurred processing %s: %s", content, err.Error())
				s.ChannelMessageSend(m.ChannelID, err.Error())
			} else {
				log.Printf("Responded ok to %s", m.Content)
				s.ChannelMessageSend(m.ChannelID, response)
			}
		} else {
			// Handle '!help', '!license', '!source'
			switch content[0] {
			case "!help":
				if len(content[1:]) == 0 || bot.Commands["!"+content[1]] == nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Available commands:\n`%s`,`!license`,`!source`\n\n(For more information type asking for `!help <command name>`)", strings.Join(bot.CommandList, "`,`")))
				} else {
					s.ChannelMessageSend(m.ChannelID, (*bot.Commands["!"+content[1]]).Help())
				}

			case "!license":
				s.ChannelMessageSend(m.ChannelID, "https://spdx.org/licenses/OSL-3.0.html")

			case "!source":
				s.ChannelMessageSend(m.ChannelID, "https://github.com/Quozlet/BirbBot")

			default:
				log.Printf("Unrecognized command: %s", m.Content)
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Unrecognized command: `%s`", content[0]))
			}
		}

	})
	err = session.Open()
	if err != nil {
		log.Println("Failed to open WebSocket connection to Discord servers")
		return nil, err
	}
	log.Println("Opened WebSocket connection to Discord")
	return session, nil
}
