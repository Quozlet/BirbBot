package app

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
	"quozlet.net/birbbot/app/commands/noargs"
	"quozlet.net/birbbot/app/commands/noargs/animal"
	"quozlet.net/birbbot/app/commands/persistent"
	"quozlet.net/birbbot/app/commands/persistent/weather"
	"quozlet.net/birbbot/app/commands/recurring"
	"quozlet.net/birbbot/app/commands/simple"
)

var recurringCommands map[recurring.Frequency][]*RecurringCommand = map[recurring.Frequency][]*RecurringCommand{}

// Start a Discord session for a given token
func Start(secret string, dbPool *pgxpool.Pool, ticker *Timers) (*discordgo.Session, error) {
	if len(secret) == 0 {
		return nil, errors.New("Not attempting connection, secret seems incorrect")
	}
	commandMap, commandList := discoverCommand(dbPool)
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
		commandHandler(s, m, dbPool, commandMap, commandList)
	})
	ticker.Start(recurringCommands, dbPool, session)
	if err = session.Open(); err != nil {
		log.Println("Failed to open WebSocket connection to Discord servers")
		return nil, err
	}
	log.Println("Opened WebSocket connection to Discord")
	return session, nil
}

func commandHandler(s *discordgo.Session, m *discordgo.MessageCreate, dbPool *pgxpool.Pool, commandMap map[string]*Command, commandList []string) {
	content := strings.Fields(strings.ToLower(m.Content))
	cmd := commandMap[content[0]]
	log.Printf("Ack %s: %s", m.Author.Username, m.Content)
	responses := func() []string {
		if cmd != nil {
			if err := s.MessageReactionAdd(m.ChannelID, m.Message.ID, "✅"); err != nil {
				log.Println(err)
			}
			defer func() {
				if err := s.MessageReactionRemove(m.ChannelID, m.Message.ID, "✅", s.State.User.ID); err != nil {
					log.Println(err)
				}
			}()
			response, msgError := processMessage(m, cmd, dbPool)
			if msgError != nil {
				log.Printf("An error occurred processing %s: %s", content, msgError.Error())
				if err := s.MessageReactionRemove(m.ChannelID, m.Message.ID, "✅", s.State.User.ID); err != nil {
					log.Println(err)
				}
				if err := s.MessageReactionAdd(m.ChannelID, m.Message.ID, "❗"); err != nil {
					log.Println(err)
				}
				return []string{msgError.Error()}
			}
			log.Printf("Responded ok to %s: %s", m.Author.Username, m.Content)
			return response

		}
		// Handle '!help', '!license', '!source'
		switch content[0] {
		case "!help":
			if len(content[1:]) == 0 || commandMap["!"+content[1]] == nil {
				return []string{fmt.Sprintf("Available commands:\n`%s`,"+
					" `!license` (the software license that applies to this bot's source code),"+
					" `!source` (a link to this bot's source code)\n\n"+
					"(For more information on a specific command: `!help <command name>`)", strings.Join(commandList, "`, `"))}
			}
			return []string{(*commandMap["!"+content[1]]).Help()}

		case "!license":
			return []string{"This bot's source code is licensed under the The Open Software License 3.0 (https://spdx.org/licenses/OSL-3.0.html)"}

		case "!source":
			return []string{"https://github.com/Quozlet/BirbBot"}

		default:
			log.Printf("Unrecognized command: %s", m.Content)
			return []string{fmt.Sprintf("Unrecognized command: `%s`", content[0])}
		}

	}()
	if len(responses) != 0 {
		for _, response := range responses {
			_, err := s.ChannelMessageSend(m.ChannelID, response)
			if err != nil {
				log.Printf("Failed to respond: %s", err)
			}
		}
	}

}

// TODO: Automatically populate commands (requires some AST parser black magic)
// In the meantime newly added commands must implement all methods in the Command interface and be added to the list
func discoverCommand(dbPool *pgxpool.Pool) (map[string]*Command, []string) {
	commandMap := make(map[string]*Command)
	for _, cmd := range []interface{}{
		animal.Bird{},
		animal.Cat{},
		animal.Dog{},
		noargs.Fortune{},
		noargs.FortuneCookie{},
		persistent.RSS{},
		persistent.Sub{},
		recurring.SubCheck{},
		simple.Choose{},
		simple.Cowsay{},
		simple.EightBall{},
		simple.Issue{},
		simple.Search{},
		simple.Wiki{},
		weather.Weather{},
		weather.Forecast{},
	} {
		command, ok := cmd.(Command)
		if ok && isValidCommand(&command, dbPool) {
			for _, alias := range command.CommandList() {
				if strings.HasPrefix(alias, "!") {
					commandMap[alias] = &command
				} else {
					log.Printf("Not registering %s (doesn't start with '!')", alias)
				}
			}
		} else {
			recurringCmd, isRecurring := cmd.(RecurringCommand)
			if isRecurring {
				freq := recurringCmd.Frequency()
				recurringCommands[freq] = append(recurringCommands[freq], &recurringCmd)
				log.Printf("Registered recurring command: %s", reflect.TypeOf(recurringCmd).Name())
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

func isValidCommand(command *Command, dbPool *pgxpool.Pool) bool {
	simpleCmd, isSimple := (*command).(SimpleCommand)
	noArgsCmd, hasNoArgs := (*command).(NoArgsCommand)
	persistentCmd, isPersistent := (*command).(PersistentCommand)
	commandName := reflect.TypeOf(*command).Name()
	if isSimple {
		if err := simpleCmd.Check(); err != nil {
			log.Printf("%s recognized but not registered: %s", commandName, err)
			return false
		}
	} else if hasNoArgs {
		if err := noArgsCmd.Check(); err != nil {
			log.Printf("%s recognized but not registered: %s", commandName, err)
			return false
		}
	} else if isPersistent {
		if err := persistentCmd.Check(dbPool); err != nil {
			log.Printf("%s recognized but not registered: %s", commandName, err)
			return false
		}
	} else {
		log.Printf("%s was recognized as a command, but does not implement a required interface."+
			" It is therefore ignored", reflect.TypeOf(*command).Name())
		return false
	}
	return true
}

func processMessage(m *discordgo.MessageCreate, command *Command, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	simpleCmd, isSimple := (*command).(SimpleCommand)
	noArgsCmd, hasNoArgs := (*command).(NoArgsCommand)
	persistentCmd, isPersistent := (*command).(PersistentCommand)
	if isSimple {
		return simpleCmd.ProcessMessage(m)
	} else if hasNoArgs {
		return noArgsCmd.ProcessMessage()
	} else if isPersistent {
		return persistentCmd.ProcessMessage(m, dbPool)
	} else {
		log.Fatalf("Got %s, an invalid command!", reflect.TypeOf(*command).Name())
		return nil, commands.NewError("A critical error occurred processing this message")
	}
}
