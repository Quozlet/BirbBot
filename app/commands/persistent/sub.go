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

const subTableDefinition string = "CREATE TABLE IF NOT EXISTS Subscriptions (FeedID INTEGER PRIMARY KEY REFERENCES Feeds(ID), Channel TEXT UNIQUE NOT NULL)"
const subInsert string = "INSERT INTO Subscriptions(FeedID, Channel) VALUES ($1, $2) ON CONFLICT (FeedID) DO UPDATE SET Channel=excluded.Channel"
const subList string = "SELECT Channel FROM Subscriptions WHERE FeedID=$1"

// Sub is a Command to subscribe a certain RSS feed to a channel
type Sub struct{}

// Check will assert that the Subscription table exists
func (s Sub) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), subTableDefinition)
	if err != nil {
		return err
	}
	log.Printf("Subscription: %s", tag)
	return nil
}

// ProcessMessage will create an association between an RSS feed and channel
func (s Sub) ProcessMessage(m *discordgo.MessageCreate, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) != 3 {
		return "", commands.NewError("Require 2 fields, the ID, and channel to subscribe it to")
	}
	id, err := strconv.ParseInt(splitContent[1], 0, 64)
	if err != nil {
		log.Println(err)
		return "", commands.NewError(fmt.Sprintf("%s is not a valid ID, so I can't look up a feed using it", splitContent[1]))
	}
	channelID := string([]rune(splitContent[2])[2:20])
	tag, err := dbPool.Exec(context.Background(), subInsert, id, channelID)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Failed to associate the feed with the channel." +
			" Check that the ID exists")
	}
	log.Printf("%s (actually inserted %d, %s)", tag, id, channelID)
	return fmt.Sprintf("Got it! Associated %d to %s", id, strings.Fields(m.Content)[2]), nil
}

// Help returns the help message for the RSS Command
func (s Sub) Help() string {
	return "`!sub <id> <channel>` subscribes the RSS feed identified by ID to the provided channel\n" +
		"Check `!rss list` for the list of RSS feeds and IDs\n" +
		"_Refresh rate is once per hour per feed (but only for new content, it uses the same rules as `!rss latest`)_"
}

// CommandList returns a list of aliases for the RSS Command
func (s Sub) CommandList() []string {
	return []string{"!sub"}
}
