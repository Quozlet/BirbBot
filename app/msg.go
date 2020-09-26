package app

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"quozlet.net/birbbot/app/commands"
	"quozlet.net/birbbot/app/commands/audio"
	handler "quozlet.net/birbbot/util"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jonas747/dca"
)

type discordInfo struct {
	session *discordgo.Session
	message *discordgo.MessageCreate
}

type msgInfo struct {
	handler             *Command
	msgChannel          chan commands.MessageResponse
	audioChannel        chan *audio.Data
	voiceCommandChannel chan audio.VoiceCommand
}

func processCommand(discord discordInfo, msg msgInfo, dbPool *pgxpool.Pool) {
	command := msg.handler
	msg.msgChannel <- commands.MessageResponse{
		ChannelID: discord.message.ChannelID,
		Reaction: commands.ReactionResponse{
			MessageID: discord.message.ID,
			Add:       "✅",
		},
	}

	defer func() {
		msg.msgChannel <- commands.MessageResponse{
			ChannelID: discord.message.ChannelID,
			Reaction: commands.ReactionResponse{
				MessageID: discord.message.ID,
				Remove:    "✅",
			},
		}
	}()

	if err := processMessage(dbPool, command, msg, discord); err != nil {
		log.Printf("An error occurred processing \"%s\"", discord.message.Content)
		msg.msgChannel <- commands.MessageResponse{
			ChannelID: discord.message.ChannelID,
			Reaction: commands.ReactionResponse{
				MessageID: discord.message.ID,
				Add:       "❗",
			},
		}
		msg.msgChannel <- commands.MessageResponse{
			ChannelID: discord.message.ChannelID,
			Message:   err.Error(),
		}
	}
}

func handleAudioCommandCommand(
	s *discordgo.Session,
	msg *discordgo.MessageCreate,
	cmd *AudioCommand,
	responseChannel chan<- commands.MessageResponse,
	audioChannel chan<- *audio.Data,
	voiceCommandChannel chan<- audio.VoiceCommand,
) *commands.CommandError {
	var commandError *commands.CommandError

	textChannel, err := s.State.Channel(msg.ChannelID)
	if commandError = commands.CreateCommandError(
		"Wasn't able to figure out what channel that message came from",
		err,
	); commandError != nil {
		return commandError
	}

	guild, err := s.State.Guild(textChannel.GuildID)
	if commandError = commands.CreateCommandError("Couldn't figure out what server to join", err); commandError != nil {
		return commandError
	}

	for _, voiceState := range guild.VoiceStates {
		if voiceState.UserID == msg.Author.ID {
			data, err := (*cmd).ProcessMessage(responseChannel, voiceCommandChannel, msg)
			if err != nil {
				return err
			}

			if data == nil {
				return nil
			}

			data.GuildID = textChannel.GuildID
			data.VoiceChannelID = voiceState.ChannelID
			data.TextChannelID = msg.ChannelID
			audioChannel <- data

			return nil
		}
	}

	return commands.NewError("You must be in a voice channel to play audio")
}

func processMessage(dbPool *pgxpool.Pool, command *Command, msg msgInfo, discord discordInfo) *commands.CommandError {
	simpleCmd, isSimple := (*command).(SimpleCommand)
	noArgsCmd, hasNoArgs := (*command).(NoArgsCommand)
	persistentCmd, isPersistent := (*command).(PersistentCommand)
	audioCmd, isAudio := (*command).(AudioCommand)

	switch {
	case isSimple:
		return simpleCmd.ProcessMessage(msg.msgChannel, discord.message)
	case hasNoArgs:
		responses, err := noArgsCmd.ProcessMessage()
		if err != nil {
			return err
		}

		for _, response := range responses {
			msg.msgChannel <- commands.MessageResponse{
				ChannelID: discord.message.ChannelID,
				Message:   response,
			}
		}

		return nil
	case isPersistent:
		return persistentCmd.ProcessMessage(msg.msgChannel, discord.message, dbPool)
	case isAudio:
		return handleAudioCommandCommand(discord.session,
			discord.message,
			&audioCmd,
			msg.msgChannel,
			msg.audioChannel,
			msg.voiceCommandChannel,
		)
	default:
		log.Fatalf("Got %s, an invalid command!"+
			" This is most likely from introducing a new command variant but failing to handle the interface above",
			reflect.TypeOf(*command).Name(),
		)

		return commands.NewError("A critical error occurred processing this message!!!")
	}
}

const audioActionReattempts = 20

type currentAudioContainer struct {
	voiceConnection  *discordgo.VoiceConnection
	currentlyPlaying *dca.StreamingSession
	currentData      *audio.Data
}

func waitForAudio(
	session *discordgo.Session,
	audioChannel <-chan *audio.Data,
	messageChannel chan<- commands.MessageResponse,
	voiceCommandChannel chan audio.VoiceCommand,
) {
	queue := make([]*audio.Data, 0)
	mutex := &sync.Mutex{}
	currentAudio := currentAudioContainer{}

	go controlCurrentStream(voiceCommandChannel, messageChannel, mutex, &currentAudio, &queue)
	go queueNewAudio(&queue, mutex, audioChannel)

	for {
		var err error

		currentAudio.currentData = nil

		mutex.Lock()
		if len(queue) == 0 {
			mutex.Unlock()

			continue
		}

		currentAudio.currentData = queue[0]
		mutex.Unlock()

		if currentAudio.voiceConnection == nil {
			if err := connectToVoice(session, &currentAudio, messageChannel); err != nil {
				continue
			}
		}

		done := make(chan error)

		if err := playAudio(&currentAudio, messageChannel, done); err != nil {
			continue
		}

		err = <-done
		if err != io.EOF {
			handleNonDisconnectError(&currentAudio, err, &queue, mutex)
		}

		currentAudio.currentlyPlaying = nil

		go handler.LogErrorMsg("Failed to cleanup: %s", currentAudio.currentData.Cleanup())
		mutex.Lock()
		if len(queue) != 0 {
			queue = queue[1:]
		}

		if len(queue) == 0 {
			handleEmptyQueue(&currentAudio, messageChannel)
		}
		mutex.Unlock()
		time.Sleep(time.Second)
	}
}

func queueNewAudio(queue *[]*audio.Data, mutex *sync.Mutex, audioChannel <-chan *audio.Data) {
	for audioData := range audioChannel {
		mutex.Lock()
		if len(*queue) != 0 {
			// Don't waste time caching if this is the only thing to be played
			go audioData.CacheAsFile()
		}

		*queue = append(*queue, audioData)
		mutex.Unlock()
	}
}

func handleNonDisconnectError(currentAudio *currentAudioContainer, err error, queue *[]*audio.Data, mutex *sync.Mutex) {
	handler.LogErrorMsg(
		fmt.Sprintf(
			"An error occurred while playing %s (possibly disconnected before finished). Reconnecting...",
			currentAudio.currentData.Title,
		),
		err,
	)
	mutex.Lock()
	if len(*queue) != 0 {
		*queue = (*queue)[1:]
	}

	if err := leaveVoice(currentAudio.voiceConnection); err != nil {
		log.Println(err)
	}

	currentAudio.voiceConnection = nil

	audio.SetInVoice(false)
	mutex.Unlock()
}

func playAudio(
	currentAudio *currentAudioContainer,
	messageChannel chan<- commands.MessageResponse,
	done chan error,
) error {
	handler.SendErrorMsg(
		commands.MessageResponse{
			Message:   "An error occurred when trying to speak",
			ChannelID: currentAudio.currentData.TextChannelID,
		},
		messageChannel,
		setSpeaking(currentAudio.voiceConnection, true),
	)
	messageChannel <- commands.MessageResponse{
		Message:   fmt.Sprintf("Playing \"%s\"", currentAudio.currentData.Title),
		ChannelID: currentAudio.currentData.TextChannelID,
	}
	time.Sleep(250 * time.Millisecond)

	memData, fileData, err := currentAudio.currentData.AudioSource()

	if (handler.SendErrorMsg(
		commands.MessageResponse{
			Message:   fmt.Sprintf("An error occurred trying to fetch %s", currentAudio.currentData.Title),
			ChannelID: currentAudio.currentData.TextChannelID,
		},
		messageChannel,
		err,
	)) {
		return err
	}

	if memData != nil {
		currentAudio.currentlyPlaying = dca.NewStream(memData, currentAudio.voiceConnection, done)
	} else if fileData != nil {
		currentAudio.currentlyPlaying = dca.NewStream(fileData, currentAudio.voiceConnection, done)
	}

	return nil
}

func connectToVoice(
	session *discordgo.Session,
	currentAudio *currentAudioContainer,
	messageChannel chan<- commands.MessageResponse,
) error {
	var err error
	currentAudio.voiceConnection, err = joinVoice(
		session,
		currentAudio.currentData.GuildID,
		currentAudio.currentData.VoiceChannelID,
	)
	if (handler.SendErrorMsg(
		commands.MessageResponse{
			Message:   "An error occurred trying to join voice",
			ChannelID: currentAudio.currentData.TextChannelID,
		},
		messageChannel,
		err,
	)) {
		return err
	}

	audio.SetInVoice(true)

	return nil
}

func controlCurrentStream(
	voiceCommandChannel chan audio.VoiceCommand,
	messageChannel chan<- commands.MessageResponse,
	mutex *sync.Mutex,
	currentAudio *currentAudioContainer,
	queue *[]*audio.Data,
) {
	for vc := range voiceCommandChannel {
		if currentAudio.currentlyPlaying != nil {
			switch vc {
			case audio.Leave:
				mutex.Lock()
				if currentAudio.voiceConnection != nil {
					handler.LogError(leaveVoice(currentAudio.voiceConnection))
					currentAudio.voiceConnection = nil

					audio.SetInVoice(false)
				}

				*queue = nil
				*queue = make([]*audio.Data, 0)
				mutex.Unlock()
			case audio.Start:
				currentAudio.currentlyPlaying.SetPaused(false)
			case audio.Stop:
				currentAudio.currentlyPlaying.SetPaused(true)
			case audio.List:
				mutex.Lock()
				var builder strings.Builder

				builder.WriteString(fmt.Sprintf("```\nCurrently playing: %s (currently at %s)",
					currentAudio.currentData.Title,
					currentAudio.currentlyPlaying.PlaybackPosition(),
				))

				for i, data := range *queue {
					builder.WriteString(fmt.Sprintf("\n%d: %s", i+1, data.Title))
				}

				builder.WriteString("\n```")
				messageChannel <- commands.MessageResponse{
					Message:   builder.String(),
					ChannelID: currentAudio.currentData.TextChannelID,
				}
				mutex.Unlock()
			}
		}
	}
}

func joinVoice(session *discordgo.Session, guildID string, voiceChannelID string) (*discordgo.VoiceConnection, error) {
	var vc *discordgo.VoiceConnection

	var err error

	for i := 0; i < audioActionReattempts; i++ {
		vc, err = session.ChannelVoiceJoin(guildID, voiceChannelID, false, true)
		if err != nil {
			log.Printf("Trying to join voice, attempt %d of %d: %s", i, audioActionReattempts, err)

			continue
		}

		audio.SetInVoice(true)

		return vc, nil
	}

	return nil, err
}

func leaveVoice(vc *discordgo.VoiceConnection) error {
	var err error
	for i := 0; i < audioActionReattempts; i++ {
		if err = vc.Disconnect(); err != nil {
			log.Printf("Trying to leave voice, attempt %d of %d: %s", i, audioActionReattempts, err)

			continue
		}

		audio.SetInVoice(false)

		break
	}

	return err
}

func setSpeaking(vc *discordgo.VoiceConnection, speaking bool) error {
	var err error
	for i := 0; i < audioActionReattempts; i++ {
		if err = vc.Speaking(speaking); err != nil {
			log.Printf("Set speaking to %t, attempt %d of %d: %s", speaking, i, audioActionReattempts, err)

			continue
		}

		break
	}

	return err
}

func handleEmptyQueue(currentAudio *currentAudioContainer, messageChannel chan<- commands.MessageResponse) {
	if currentAudio.voiceConnection != nil {
		handler.SendErrorMsg(commands.MessageResponse{
			Message:   "Unable to stop speaking, probably was forcibly disconnected",
			ChannelID: currentAudio.currentData.TextChannelID,
		}, messageChannel, setSpeaking(currentAudio.voiceConnection, false))
		handler.SendErrorMsg(commands.MessageResponse{
			Message:   "Nothing more in the queue, but I can't leave",
			ChannelID: currentAudio.currentData.TextChannelID,
		}, messageChannel, leaveVoice(currentAudio.voiceConnection))

		currentAudio.voiceConnection = nil

		audio.SetInVoice(false)
	}
}
