package app

import (
	"fmt"
	"io"
	"log"
	"reflect"
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
	simpleCmd, isSimple := (*command).(SimpleCommand)
	noArgsCmd, hasNoArgs := (*command).(NoArgsCommand)
	persistentCmd, isPersistent := (*command).(PersistentCommand)
	audioCmd, isAudio := (*command).(AudioCommand)
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
	if err := func() *commands.CommandError {
		if isSimple {
			return simpleCmd.ProcessMessage(msg.msgChannel, discord.message)
		} else if hasNoArgs {
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
		} else if isPersistent {
			return persistentCmd.ProcessMessage(msg.msgChannel, discord.message, dbPool)
		} else if isAudio {
			return handleAudioCommandCommand(discord.session, discord.message, &audioCmd, msg.msgChannel, msg.audioChannel, msg.voiceCommandChannel)
		} else {
			log.Fatalf("Got %s, an invalid command!"+
				" This is most likely from introducing a new command variant but failing to handle the interface above", reflect.TypeOf(*command).Name())
			return commands.NewError("A critical error occurred processing this message!!!")
		}
	}(); err != nil {
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
	if commandError = commands.CreateCommandError("Wasn't able to figure out what channel that message came from", err); commandError != nil {
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

type currentAudioContainer struct {
	voiceConnection  *discordgo.VoiceConnection
	currentlyPlaying *dca.StreamingSession
}

func waitForAudio(session *discordgo.Session, audioChannel <-chan *audio.Data, messageChannel chan<- commands.MessageResponse, voiceCommandChannel chan audio.VoiceCommand) {
	queue := make([]*audio.Data, 0)
	mutex := &sync.Mutex{}
	currentAudio := currentAudioContainer{}
	go controlCurrentStream(voiceCommandChannel, mutex, &currentAudio, &queue)
	go queueNewAudio(&queue, mutex, audioChannel)
	for {
		var err error
		var currentData *audio.Data
		mutex.Lock()
		if len(queue) == 0 {
			mutex.Unlock()
			continue
		}
		currentData, queue = queue[0], queue[1:]
		mutex.Unlock()
		if currentAudio.voiceConnection == nil {
			currentAudio.voiceConnection, err = session.ChannelVoiceJoin(currentData.GuildID, currentData.VoiceChannelID, false, true)
			if (handler.SendErrorMsg(
				commands.MessageResponse{
					Message:   "An error occurred trying to join voice",
					ChannelID: currentData.TextChannelID,
				},
				messageChannel,
				err,
			)) {
				continue
			}
			audio.SetInVoice(true)
		}
		if (handler.SendErrorMsg(
			commands.MessageResponse{
				Message:   "An error occurred trying to speak, skipping",
				ChannelID: currentData.TextChannelID,
			},
			messageChannel,
			currentAudio.voiceConnection.Speaking(true),
		)) {
			continue
		}
		messageChannel <- commands.MessageResponse{
			Message:   fmt.Sprintf("Playing \"%s\"", currentData.Title),
			ChannelID: currentData.TextChannelID,
		}
		time.Sleep(250 * time.Millisecond)
		done := make(chan error)
		memData, fileData, err := currentData.AudioSource()
		if (handler.SendErrorMsg(
			commands.MessageResponse{
				Message:   fmt.Sprintf("An error occurred trying to fetch %s", currentData.Title),
				ChannelID: currentData.TextChannelID,
			},
			messageChannel,
			err,
		)) {
			continue
		}
		if memData != nil {
			currentAudio.currentlyPlaying = dca.NewStream(memData, currentAudio.voiceConnection, done)
		} else if fileData != nil {
			currentAudio.currentlyPlaying = dca.NewStream(fileData, currentAudio.voiceConnection, done)
		}
		err = <-done
		if err != io.EOF {
			handler.SendErrorMsg(commands.MessageResponse{
				Message:   fmt.Sprintf("An error occurred while playing %s", currentData.Title),
				ChannelID: currentData.TextChannelID,
			}, messageChannel, err)
		}
		currentAudio.currentlyPlaying = nil
		go handler.LogErrorMsg("Failed to cleanup: %s", currentData.Cleanup())
		mutex.Lock()
		if len(queue) == 0 {
			if currentAudio.voiceConnection != nil {
				handler.SendErrorMsg(commands.MessageResponse{
					Message:   "Unable to stop speaking, probably was forcibly disconnected",
					ChannelID: currentData.TextChannelID,
				}, messageChannel, currentAudio.voiceConnection.Speaking(false))
				handler.SendErrorMsg(commands.MessageResponse{
					Message:   "Nothing more in the queue, but I can't leave",
					ChannelID: currentData.TextChannelID,
				}, messageChannel, currentAudio.voiceConnection.Disconnect())
				audio.SetInVoice(false)
				currentAudio.voiceConnection = nil
			}
		}
		mutex.Unlock()
		time.Sleep(time.Second)
	}
}

func queueNewAudio(queue *[]*audio.Data, mutex *sync.Mutex, audioChannel <-chan *audio.Data) {
	for audioData := range audioChannel {
		mutex.Lock()
		if len(*queue) != 0 {
			// Don't waste time caching if this is the only thing to be played next
			go audioData.CacheAsFile()
		}
		*queue = append(*queue, audioData)
		mutex.Unlock()
	}
}

func controlCurrentStream(
	voiceCommandChannel chan audio.VoiceCommand,
	mutex *sync.Mutex,
	currentAudio *currentAudioContainer,
	queue *[]*audio.Data,
) {
	for vc := range voiceCommandChannel {
		if (*currentAudio).currentlyPlaying != nil {
			switch vc {
			case audio.Leave:
				mutex.Lock()
				if (*currentAudio).voiceConnection != nil {
					handler.LogError((*currentAudio).voiceConnection.Disconnect())
					audio.SetInVoice(false)
					(*currentAudio).voiceConnection = nil
				}
				*queue = nil
				*queue = make([]*audio.Data, 0)
				mutex.Unlock()
			case audio.Start:
				(*currentAudio).currentlyPlaying.SetPaused(false)
			case audio.Stop:
				(*currentAudio).currentlyPlaying.SetPaused(true)
			}
		}
	}
}
