package app

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	handler "quozlet.net/birbbot/util"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
	"quozlet.net/birbbot/app/commands/audio"
	"quozlet.net/birbbot/app/commands/noargs"
	"quozlet.net/birbbot/app/commands/noargs/animal"
	"quozlet.net/birbbot/app/commands/persistent"
	"quozlet.net/birbbot/app/commands/persistent/weather"
	"quozlet.net/birbbot/app/commands/recurring"
	"quozlet.net/birbbot/app/commands/simple"
)

var (
	recurringCommands  = map[recurring.Frequency][]*RecurringCommand{}
	errIncorrectSecret = errors.New("not attempting connection, secret seems incorrect")
)

// Start a Discord session for a given token.
func Start(secret string, dbPool *pgxpool.Pool, ticker *Timers) (*discordgo.Session, error) {
	if len(secret) == 0 {
		return nil, errIncorrectSecret
	}

	commandMap, commandList := discoverCommand(dbPool)

	session, err := discordgo.New("Bot " + secret)
	if err != nil {
		log.Println("Unable to create Discord session")

		return nil, err
	}

	log.Println("Bot Token accepted by Discord, beginning connection...")

	messageChannel := make(chan commands.MessageResponse)

	go waitForCommandResponses(session, messageChannel)

	audioChannel := make(chan *audio.Data)
	voiceCommandChannel := make(chan audio.VoiceCommand)

	go waitForAudio(session, audioChannel, messageChannel, voiceCommandChannel)
	// TODO: If panicking while processing a command, error instead of crashing
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages without the '!' prefix or with own ID
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, string(Prefix)) {
			return
		}
		commandHandler(s, m, dbPool, commandMap, commandList, messageChannel, audioChannel, voiceCommandChannel)
	})

	go ticker.Start(recurringCommands, dbPool, session)

	if err = session.Open(); err != nil {
		log.Println("Failed to open WebSocket connection to Discord servers")

		return nil, err
	}

	log.Println("Opened WebSocket connection to Discord...")

	return session, nil
}

func commandHandler(
	s *discordgo.Session,
	m *discordgo.MessageCreate,
	dbPool *pgxpool.Pool,
	commandMap map[string]*Command,
	commandList []string,
	msgChannel chan commands.MessageResponse,
	audioChannel chan *audio.Data,
	voiceCommandChannel chan audio.VoiceCommand,
) {
	content := strings.Fields(strings.ToLower(m.Content))
	cmd, found := commandMap[content[0]]

	log.Printf("Ack %s: %s", m.Author.Username, m.Content)

	if !found {
		if content[0] == BuildCommandName("help") {
			msgChannel <- handleHelpMessage(m.ChannelID, commandList, content[1:], commandMap)
		} else {
			log.Printf("Unrecognized command: %s", m.Content)
			msgChannel <- commands.MessageResponse{
				ChannelID: m.ChannelID,
				Message:   fmt.Sprintf("Unrecognized command: `%s`", content[0]),
			}
		}

		return
	}

	go processCommand(discordInfo{
		session: s,
		message: m,
	}, msgInfo{
		handler:             cmd,
		msgChannel:          msgChannel,
		audioChannel:        audioChannel,
		voiceCommandChannel: voiceCommandChannel,
	}, dbPool)
}

// TODO: Automatically populate commands (requires some AST parser black magic)
// In the meantime newly added commands must implement all methods in the Command interface and be added to the list.
func discoverCommand(dbPool *pgxpool.Pool) (map[string]*Command, []string) {
	commandMap := make(map[string]*Command)

	for _, cmd := range []interface{}{
		animal.Bird{},
		animal.Cat{},
		animal.Dog{},
		audio.Disconnect{},
		audio.Play{},
		audio.Pause{},
		audio.Queue{},
		noargs.Fortune{},
		noargs.FortuneCookie{},
		noargs.License{},
		noargs.Source{},
		persistent.Filter{},
		persistent.RSS{},
		persistent.Sub{},
		recurring.SubCheck{},
		recurring.SubCleanup{},
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
				commandMap[BuildCommandName(alias)] = &command
			}
		} else {
			recurringCmd, isRecurring := cmd.(RecurringCommand)
			if isRecurring {
				freq := recurringCmd.Frequency()
				recurringCommands[freq] = append(recurringCommands[freq], &recurringCmd)
				log.Printf("Registered recurring %v command: %s", recurringCmd.Frequency(), reflect.TypeOf(recurringCmd).Name())
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
	_, isAudio := (*command).(AudioCommand)
	commandName := reflect.TypeOf(*command).Name()

	switch {
	case isSimple:
		if err := simpleCmd.Check(); err != nil {
			log.Printf("%s recognized but not registered: %s", commandName, err)

			return false
		}
	case hasNoArgs:
		if err := noArgsCmd.Check(); err != nil {
			log.Printf("%s recognized but not registered: %s", commandName, err)

			return false
		}
	case isPersistent:
		if err := persistentCmd.Check(dbPool); err != nil {
			log.Printf("%s recognized but not registered: %s", commandName, err)

			return false
		}
	case isAudio:
		return true
	default:
		log.Fatalf("%s was recognized as a command, but does not implement a required interface.",
			reflect.TypeOf(*command).Name(),
		)

		return false
	}

	return true
}

func handleHelpMessage(
	channelID string,
	commandList []string,
	args []string,
	commandMap map[string]*Command,
) commands.MessageResponse {
	helpMsg := fmt.Sprintf("Available commands (All require prefix `%s`):\n`%s`,"+
		"(For more information on a specific command: `help <command name>`)",
		string(Prefix),
		strings.Join(commandList, "`, `"),
	)

	if len(args) != 0 {
		cmd, ok := commandMap[BuildCommandName(args[1])]
		if !ok {
			helpMsg = fmt.Sprintf("Cannot find help message, command `%s` does not exist", BuildCommandName(args[1]))
		} else {
			helpMsg = (*cmd).Help()
		}
	}

	return commands.MessageResponse{
		ChannelID: channelID,
		Message:   helpMsg,
	}
}

func waitForCommandResponses(session *discordgo.Session, messageChannel <-chan commands.MessageResponse) {
	for pendingMsg := range messageChannel {
		if len(pendingMsg.Reaction.MessageID) != 0 {
			if len(pendingMsg.Reaction.Add) != 0 {
				handler.LogErrorMsg(
					fmt.Sprintf("Failed to add reaction %s", pendingMsg.Reaction.Add),
					session.MessageReactionAdd(
						pendingMsg.ChannelID,
						pendingMsg.Reaction.MessageID,
						pendingMsg.Reaction.Add,
					),
				)
			}

			if len(pendingMsg.Reaction.Remove) != 0 {
				handler.LogErrorMsg(
					fmt.Sprintf("Failed to remove reaction %s", pendingMsg.Reaction.Remove),
					session.MessageReactionRemove(
						pendingMsg.ChannelID,
						pendingMsg.Reaction.MessageID,
						pendingMsg.Reaction.Remove,
						session.State.User.ID,
					),
				)
			}
		}

		if len(pendingMsg.Message) != 0 {
			_, err := session.ChannelMessageSend(pendingMsg.ChannelID, pendingMsg.Message)
			handler.LogError(err)
		}
	}
}
