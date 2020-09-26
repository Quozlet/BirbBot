package persistent

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
	"quozlet.net/birbbot/app/commands"
)

const (
	invalidRSSIDErrorMsg = "Hey, so, uh, I need an _ID_, a number. %s is not a number"
	missingRSSIDErrorMsg = "I understood the ID, but the database says it's invalid. Can you double-check?"
)

// TODO: Finalize schema with RSS/Sub combined w/ proper handling of CASCADE etc.
const (
	rssTableDefinition string = "CREATE TABLE IF NOT EXISTS Feeds " +
		"(ID SERIAL PRIMARY KEY, Title TEXT NOT NULL, URL TEXT UNIQUE NOT NULL, LastItems JSONB NOT NULL)"
	rssNewFeed string = "INSERT INTO Feeds(Title, URL, LastItems) VALUES ($1, $2, $3) ON CONFLICT (URL) DO NOTHING"
	rssList    string = "SELECT ID, Title, URL, LastItems FROM Feeds ORDER BY ID"
)

// RSSSelect an RSS feed by ID.
const RSSSelect string = "SELECT Title, URL, LastItems FROM Feeds WHERE ID = $1"

// RSSUpdateLastItem with a new hash.
const RSSUpdateLastItem string = "UPDATE Feeds SET LastItems = $1 WHERE ID = $2"

// RSS is a command to fetch an RSS feed for validation.
type RSS struct{}

// Check returns nil.
func (r RSS) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), rssTableDefinition)
	if err != nil {
		return err
	}

	log.Printf("RSS: %s", tag)

	return nil
}

// ProcessMessage attempts to parse the first argument as a URL to an RSS feed,
// then fetch the first argument. If any step fails, an error is returned.
func (r RSS) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return commands.NewError("Sure, let me test if that's valid.\n" +
			"Here comes the feed: _You are a horrible person_. " +
			"I'm serious, that's what's in the feed: _\"A horrible person\"_." +
			" We weren't even testing for that")
	}

	message := splitContent[1:]

	switch message[0] {
	case "list":
		return listFeeds(response, m.ChannelID, dbPool)

	case "find":
		return findFeedByID(response, m.ChannelID, message, dbPool)

	case "latest":
		return fetchLatest(response, m.ChannelID, message, dbPool)

	default:
		return storeNewFeed(response, m.ChannelID, message[0], dbPool)
	}
}

func listFeeds(
	response chan<- commands.MessageResponse,
	channelID string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	feeds, err := selectAllFeedDB(dbPool)

	if commandError = commands.CreateCommandError(
		"Couldn't get a list of feeds from the database. "+
			"Try again later",
		err,
	); commandError != nil {
		return commandError
	}

	builder := strings.Builder{}

	for _, info := range feeds {
		builder.WriteString(fmt.Sprintf("ID: %d | %s (%s)\n", info.ID, info.Title, info.URL))
	}

	if builder.Len() == 0 {
		return commands.NewError("Can't list, you haven't subscribed to any feeds yet")
	}
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message:   builder.String(),
	}

	return nil
}

func findFeedByID(
	response chan<- commands.MessageResponse,
	channelID string,
	args []string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	if len(args) == 1 {
		return commands.NewError("<insert 404 joke here> Look, you didn't provide anything to find")
	}

	id, err := strconv.ParseInt(args[1], 0, 64)

	if commandError = commands.CreateCommandError(
		fmt.Sprintf(invalidRSSIDErrorMsg, args[1]),
		err,
	); commandError != nil {
		return commandError
	}

	info, err := selectFeedDB(dbPool, id)

	if commandError = commands.CreateCommandError(
		missingRSSIDErrorMsg,
		err,
	); commandError != nil {
		return commandError
	}
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message:   fmt.Sprintf("**%s** <%s>", info.Title, info.URL),
	}

	return nil
}

func fetchLatest(
	response chan<- commands.MessageResponse,
	channelID string,
	args []string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	if len(args) == 1 {
		return commands.NewError("**My Database**\nzilch\n\nTry providing an ID to search by")
	}

	id, err := strconv.ParseInt(args[1], 0, 64)

	if commandError = commands.CreateCommandError(
		fmt.Sprintf(invalidRSSIDErrorMsg, args[1]),
		err,
	); commandError != nil {
		return commandError
	}

	info, err := selectFeedDB(dbPool, id)

	if commandError = commands.CreateCommandError(
		missingRSSIDErrorMsg,
		err,
	); commandError != nil {
		return commandError
	}

	url, err := url.Parse(info.URL)

	if commandError = commands.CreateCommandError(
		"This isn't good. Somehow an invalid feed URL was saved into the database for this ID",
		err,
	); commandError != nil {
		return commandError
	}

	feed, err := RefreshFeed(url)

	if commandError = commands.CreateCommandError(
		"Tried to fetch the feed, but some error occurred reading it",
		err,
	); commandError != nil {
		return commandError
	}

	urls, haveNewFeeds := deduplicateItems(feed.Items, FetchRegex(id, dbPool), info, response, channelID)

	if !haveNewFeeds {
		return nil
	}

	if commandError = updateLastItem(dbPool, urls, info.LastItems, id); commandError != nil {
		return commandError
	}

	return nil
}

func storeNewFeed(
	response chan<- commands.MessageResponse,
	channelID string,
	userMsg string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	url, err := url.Parse(userMsg)
	if commandError = commands.CreateCommandError(
		fmt.Sprintf("%s doesn't seem to be a valid URL", userMsg),
		err,
	); commandError != nil {
		return commandError
	}

	url.Scheme = "https"
	feed, err := RefreshFeed(url)

	if commandError = commands.CreateCommandError(
		"Tried to fetch the feed, but some error occurred reading it",
		err,
	); commandError != nil {
		return commandError
	}

	feed.Title = html2text.HTML2Text(feed.Title)
	response <- newFeedAckResponse(feed, channelID)

	existing := make(map[string]struct{})

	for _, item := range ReduceItem(feed.Items, nil) {
		existing[item.Description] = struct{}{}
	}

	tag, err := dbPool.Exec(context.Background(), rssNewFeed, html2text.HTML2Text(feed.Title), url.String(), existing)
	if commandError = commands.CreateCommandError(
		"Went to insert this feed into the database for later, and it didn't seem to like that."+
			" Maybe provide a less spicy feed? Or try some Pepto-Bismol",
		err,
	); commandError != nil {
		return commandError
	}

	log.Printf("RSS: %s (actually inserted row with Title %s, URL %s, and %d Existing items at insertion time)", tag,
		html2text.HTML2Text(feed.Title),
		url.String(),
		len(existing))

	return nil
}

func updateLastItem(dbPool *pgxpool.Pool, urls, lastItems map[string]struct{}, id int64) *commands.CommandError {
	tag, err := dbPool.Exec(context.Background(), RSSUpdateLastItem, func() map[string]struct{} {
		for desc := range lastItems {
			urls[desc] = struct{}{}
		}

		return urls
	}(), id)
	if commandError := commands.CreateCommandError(
		"Internal errors."+
			" Couldn't save these new items as posted."+
			" They may be reposted.",
		err,
	); commandError != nil {
		return commandError
	}

	log.Printf("RSS: %s (actually inserted %d items for %d)", tag, len(urls), id)

	return nil
}

func newFeedAckResponse(feed *gofeed.Feed, channelID string) commands.MessageResponse {
	if len(feed.Description) != 0 {
		return commands.MessageResponse{
			ChannelID: channelID,
			Message: fmt.Sprintf("Fetched **%s** _(%s)_",
				feed.Title,
				html2text.HTML2Text(feed.Description)),
		}
	}

	return commands.MessageResponse{
		ChannelID: channelID,
		Message:   fmt.Sprintf("Fetched **%s**", feed.Title),
	}
}

// CommandList returns a list of aliases for the RSS Command.
func (r RSS) CommandList() []string {
	return []string{"rss"}
}

// Help returns the help message for the RSS Command.
func (r RSS) Help() string {
	return "Subscribes to an RSS feed\n" +
		"- `rss list` lists all subscribed RSS feeds\n" +
		"- `rss find <id>` finds an RSS feed by it's numerical ID\n" +
		"- `rss latest <id>` re-fetches the latest element of the feed and (if it hasn't already been posted) posts it"
}

// RSSInfo contains the posted information for a RSS feed item.
type RSSInfo struct {
	Title       string
	Description string
}

// ReduceItem reduces a list of RSS items to a list of titles and secondary text (usually URLs).
func ReduceItem(items []*gofeed.Item, regex *regexp.Regexp) []RSSInfo {
	infoItems := []RSSInfo{}

	for _, item := range items {
		rssInfo := RSSInfo{
			Title: html2text.HTML2Text(item.Title),
			Description: func() string {
				if regex != nil {
					return extractDescriptionForRegex(regex, item)
				}

				return extractDescription(item)
			}(),
		}
		if rssInfo.Description != "" {
			infoItems = append(infoItems, rssInfo)
		}
	}

	return infoItems
}

func extractDescription(item *gofeed.Item) string {
	secondary := item.Link
	if len(secondary) == 0 {
		if len(item.Enclosures) != 0 {
			return item.Enclosures[0].URL
		}

		if len(item.Description) != 0 {
			return html2text.HTML2Text(item.Description)
		}

		return html2text.HTML2Text(item.Content)
	}

	return secondary
}

func extractDescriptionForRegex(regex *regexp.Regexp, item *gofeed.Item) string {
	if regex.MatchString(item.Link) {
		return regex.FindString(item.Link)
	}

	if len(item.Enclosures) != 0 && regex.MatchString(item.Enclosures[0].URL) {
		return regex.FindString(item.Enclosures[0].URL)
	}

	if regex.MatchString(item.Description) {
		return regex.FindString(item.Description)
	}

	if regex.MatchString(item.Content) {
		return regex.FindString(item.Content)
	}

	return ""
}

func deduplicateItems(
	feedItems []*gofeed.Item,
	regex *regexp.Regexp,
	info *feedInfo,
	response chan<- commands.MessageResponse,
	channelID string,
) (map[string]struct{}, bool) {
	items := ReduceItem(feedItems, regex)
	urls := make(map[string]struct{})
	haveNewFeeds := false

	for _, item := range items {
		_, contained := info.LastItems[item.Description]
		if !contained {
			haveNewFeeds = true
			response <- commands.MessageResponse{
				ChannelID: channelID,
				Message: fmt.Sprintf("%s: **%s**\n%s",
					info.Title,
					item.Title,
					item.Description),
			}

			urls[item.Description] = struct{}{}
		}
	}

	if !haveNewFeeds {
		response <- commands.MessageResponse{
			ChannelID: channelID,
			Message:   "Nothing new to report",
		}
	}

	return urls, haveNewFeeds
}

// RefreshFeed fetches a given RSS feed.
func RefreshFeed(url *url.URL) (*gofeed.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fp := gofeed.NewParser()

	return fp.ParseURLWithContext(url.String(), ctx)
}

// SQL Helpers.

type feedInfo struct {
	ID        int64
	Title     string
	URL       string
	LastItems map[string]struct{}
}

func selectAllFeedDB(dbPool *pgxpool.Pool) ([]*feedInfo, error) {
	rows, err := dbPool.Query(context.Background(), rssList)
	if err != nil {
		return nil, err
	}

	info := []*feedInfo{}

	for rows.Next() {
		var id int64

		var title string

		var url string

		var lastItems map[string]struct{}
		if err := rows.Scan(&id, &title, &url, &lastItems); err != nil {
			return nil, err
		}

		info = append(info, &feedInfo{
			ID:        id,
			Title:     title,
			URL:       url,
			LastItems: lastItems,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return info, nil
}

func selectFeedDB(dbPool *pgxpool.Pool, id int64) (*feedInfo, error) {
	var title string

	var url string

	var lastItems map[string]struct{}
	if err := dbPool.QueryRow(context.Background(), RSSSelect, id).Scan(&title, &url, &lastItems); err != nil {
		return nil, err
	}

	return &feedInfo{ID: id, Title: title, URL: url, LastItems: lastItems}, nil
}
