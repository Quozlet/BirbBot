package persistent

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"quozlet.net/birbbot/app/commands"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	subTableDefinition string = "CREATE TABLE IF NOT EXISTS " +
		"Subscriptions (FeedID INTEGER PRIMARY KEY REFERENCES Feeds(ID), Channel TEXT NOT NULL)"
	subInsert string = "INSERT INTO Subscriptions(FeedID, Channel) VALUES ($1, $2) " +
		"ON CONFLICT (FeedID) DO UPDATE SET Channel=excluded.Channel"
	subList string = "SELECT FeedID, Channel FROM Subscriptions"
)

// Sub is a Command to subscribe a certain RSS feed to a channel.
type Sub struct{}

// Check will assert that the Subscription table exists.
func (s Sub) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), subTableDefinition)
	if err != nil {
		return err
	}

	log.Printf("Subscription: %s", tag)

	return nil
}

// ProcessMessage will create an association between an RSS feed and channel.
func (s Sub) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return commands.NewError("`sub` requires arguments")
	}

	message := splitContent[1:]

	switch message[0] {
	case "list":
		return listSubscriptions(dbPool, response, m.ChannelID)

	default:
		id, err := strconv.ParseInt(splitContent[1], 0, 64)
		if commandError = commands.CreateCommandError(
			fmt.Sprintf("%s is not a valid ID, so I can't look up a feed using it", splitContent[1]),
			err,
		); commandError != nil {
			return commandError
		}

		channelID := string([]rune(splitContent[2])[2:20])
		tag, err := dbPool.Exec(context.Background(), subInsert, id, channelID)

		if commandError = commands.CreateCommandError(
			"Failed to associate the feed with the channel."+
				" Check that the ID exists",
			err,
		); commandError != nil {
			return commandError
		}

		log.Printf("Sub: %s (actually inserted %d, %s)", tag, id, channelID)
		response <- commands.MessageResponse{
			ChannelID: m.ChannelID,
			Message:   fmt.Sprintf("Got it! Associated %d to %s", id, strings.Fields(m.Content)[2]),
		}

		return nil
	}
}

// Help returns the help message for the RSS Command.
func (s Sub) Help() string {
	return "`sub <id> <channel>` subscribes the RSS feed identified by ID to the provided channel\n" +
		"_Check `rss list` for the list of RSS feeds and IDs_\n\n" +
		"- `sub list` lists all active subscriptions\n" +
		"\n_Refresh rate is once per 30 minutes per feed (but only for new content, it uses the same rules as `rss latest`)_"
}

// CommandList returns a list of aliases for the RSS Command.
func (s Sub) CommandList() []string {
	return []string{"sub"}
}

func listSubscriptions(
	dbPool *pgxpool.Pool,
	response chan<- commands.MessageResponse,
	channelID string,
) *commands.CommandError {
	var commandError *commands.CommandError

	rows, err := dbPool.Query(context.Background(), subList)
	if commandError = commands.CreateCommandError(
		"Couldn't read a list of subscriptions from the database!",
		err,
	); commandError != nil {
		return commandError
	}

	haveActiveSubs := false

	for rows.Next() {
		var channel string

		var id int64
		if commandError = commands.CreateCommandError(
			"An error occurred reading a certain subscription's information. Aborting",
			rows.Scan(&id, &channel),
		); commandError != nil {
			return commandError
		}

		haveActiveSubs = true
		response <- commands.MessageResponse{
			ChannelID: channelID,
			Message:   fmt.Sprintf("%d -> <#%s>", id, channel),
		}
	}

	if commandError = commands.CreateCommandError(
		"An error occurred fetching the subscriptions",
		rows.Err(),
	); commandError != nil {
		return commandError
	}

	if !haveActiveSubs {
		response <- commands.MessageResponse{
			ChannelID: channelID,
			Message:   "No subscriptions are currently active",
		}
	}

	return nil
}
