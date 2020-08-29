package app

import (
	"log"
	"reflect"

	"quozlet.net/birbbot/app/commands"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

type discordInfo struct {
	session *discordgo.Session
	message *discordgo.MessageCreate
}

type msgInfo struct {
	handler *Command
	channel chan commands.MessageResponse
}

func processCommand(discord discordInfo, msg msgInfo, dbPool *pgxpool.Pool) {
	command := msg.handler
	simpleCmd, isSimple := (*command).(SimpleCommand)
	noArgsCmd, hasNoArgs := (*command).(NoArgsCommand)
	persistentCommand, isPersistent := (*command).(PersistentCommand)
	msg.channel <- commands.MessageResponse{
		ChannelID: discord.message.ChannelID,
		Reaction: commands.ReactionResponse{
			MessageID: discord.message.ID,
			Add:       "✅",
		},
	}
	defer func() {
		msg.channel <- commands.MessageResponse{
			ChannelID: discord.message.ChannelID,
			Reaction: commands.ReactionResponse{
				MessageID: discord.message.ID,
				Remove:    "✅",
			},
		}
	}()
	if err := func() *commands.CommandError {
		if isSimple {
			return simpleCmd.ProcessMessage(msg.channel, discord.message)
		} else if hasNoArgs {
			responses, err := noArgsCmd.ProcessMessage()
			if err != nil {
				return err
			}
			for _, response := range responses {
				msg.channel <- commands.MessageResponse{
					ChannelID: discord.message.ChannelID,
					Message:   response,
				}
			}
			return nil
		} else if isPersistent {
			return persistentCommand.ProcessMessage(msg.channel, discord.message, dbPool)
		} else {
			log.Fatalf("Got %s, an invalid command!"+
				" This is most likely from introducing a new command variant but failing to handle the interface above", reflect.TypeOf(*command).Name())
			return commands.NewError("A critical error occurred processing this message!!!")
		}
	}(); err != nil {
		log.Printf("An error occurred processing \"%s\"", discord.message.Content)
		msg.channel <- commands.MessageResponse{
			ChannelID: discord.message.ChannelID,
			Reaction: commands.ReactionResponse{
				MessageID: discord.message.ID,
				Add:       "❗",
			},
		}
		msg.channel <- commands.MessageResponse{
			ChannelID: discord.message.ChannelID,
			Message:   err.Error(),
		}
	}
}
