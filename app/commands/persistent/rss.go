package persistent

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
	"quozlet.net/birbbot/app/commands"
)

// TODO: Finalize schema with RSS/Sub combined w/ proper handling of CASCADE etc.
const rssTableDefinition string = "CREATE TABLE IF NOT EXISTS Feeds (ID SERIAL PRIMARY KEY, Title TEXT NOT NULL, URL TEXT UNIQUE NOT NULL, LastItems JSONB NOT NULL)"
const rssNewFeed string = "INSERT INTO Feeds(Title, URL, LastItems) VALUES ($1, $2, $3) ON CONFLICT (URL) DO NOTHING"
const rssList string = "SELECT ID, Title, URL, LastItems FROM Feeds ORDER BY ID"

// RSSSelect an RSS feed by ID
const RSSSelect string = "SELECT Title, URL, LastItems FROM Feeds WHERE ID = $1"

// DB Migration
const rssTableMigrationDrop string = "ALTER TABLE Feeds DROP COLUMN LastItemHash CASCADE"
const rssTableMigrationAdd string = "ALTER TABLE Feeds ADD COLUMN LastItems JSONB NOT NULL DEFAULT '{}'"

// RSSUpdateLastItem with a new hash
const RSSUpdateLastItem string = "UPDATE Feeds SET LastItems = $1 WHERE ID = $2"

// RSS is a command to fetch an RSS feed for validation
type RSS struct{}

// Check returns nil
func (r RSS) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), rssTableDefinition)
	if err != nil {
		return err
	}
	drop, err := dbPool.Exec(context.Background(), rssTableMigrationDrop)
	if err != nil {
		log.Printf("WARNING! Couldn't drop the table: %s", rssTableMigrationDrop)
		return err
	}
	log.Println(drop)
	add, err := dbPool.Exec(context.Background(), rssTableMigrationAdd)
	if err != nil {
		log.Printf("WARNING! Couldn't add the title: %s", rssTableMigrationAdd)
		return err
	}
	log.Println(add)
	log.Printf("RSS: %s", tag)
	return nil
}

// ProcessMessage attempts to parse the first argument as a URL to an RSS feed, then fetch the first argument. If any step fails, an error is returned
func (r RSS) ProcessMessage(m *discordgo.MessageCreate, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return nil, commands.NewError("Sure, let me test if that's valid.\n" +
			"Here comes the feed: _You are a horrible person_. " +
			"I'm serious, that's what's in the feed: _\"A horrible person\"_." +
			" We weren't even testing for that")
	}
	message := strings.Fields(strings.ToLower(m.Content))[1:]
	switch message[0] {
	case "list":
		return listFeeds(dbPool)

	case "find":
		return findFeedByID(message, dbPool)

	case "latest":
		return fetchLatest(message, dbPool)

	default:
		return storeNewFeed(message[0], dbPool)
	}
}

func listFeeds(dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	feeds, err := selectAllFeedDB(dbPool)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Couldn't get a list of feeds from the database. " +
			"Try again later")
	}
	builder := strings.Builder{}
	for _, info := range feeds {
		builder.WriteString(fmt.Sprintf("ID: %d | %s\n", info.ID, info.Title))
	}
	if builder.Len() == 0 {
		return nil, commands.NewError("Can't list, you haven't subscribed to any feeds yet")
	}
	return []string{builder.String()}, nil
}

func findFeedByID(args []string, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	if len(args) == 1 {
		return nil, commands.NewError("<insert 404 joke here> Look, you didn't provide anything to find")
	}
	id, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError(fmt.Sprintf(
			"Hey, so, uh, I need an _ID_, a number."+
				" %s is not a number", args[1]))
	}
	info, err := selectFeedDB(dbPool, id)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("I understood the ID, but the database says it's invalid." +
			" Can you double-check?")
	}
	return []string{fmt.Sprintf("**%s** <%s>", info.Title, info.URL)}, nil
}

func fetchLatest(args []string, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	if len(args) == 1 {
		return nil, commands.NewError("**My Database**\nzilch\n\nTry providing an ID to search by")
	}
	id, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError(fmt.Sprintf(
			"Hey, so, uh, I need an _ID_, a number."+
				" %s is not a number", args[1]))
	}
	info, err := selectFeedDB(dbPool, id)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("I understood the ID, but the database says it's invalid. " +
			"Can you double-check?")
	}
	url, err := url.Parse(info.URL)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("This isn't good. Somehow an invalid feed URL was saved into the database for this ID")
	}
	feed, err := RefreshFeed(url)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Tried to fetch the feed, but some error occurred reading it")
	}
	items := ReduceItem(feed.Items)
	urls := make(map[string]struct{})
	newFeeds := []string{}
	for _, item := range items {
		_, contained := info.LastItems[item.Description]
		if !contained {
			newFeeds = append(newFeeds, fmt.Sprintf("%s: **%s**\n%s", info.Title, item.Title, item.Description))
		}
		urls[item.Description] = struct{}{}
	}
	if len(newFeeds) == 0 {
		return []string{"Nothing new to report"}, nil
	}
	tag, err := dbPool.Exec(context.Background(), RSSUpdateLastItem, urls, id)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Internal errors." +
			" Couldn't save these new items as posted." +
			" They may be reposted.")
	}
	log.Println(tag)
	return newFeeds, nil
}

func storeNewFeed(userMsg string, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	url, err := url.Parse(userMsg)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError(fmt.Sprintf("%s doesn't seem to be a valid URL", userMsg))
	}
	url.Scheme = "https"
	feed, err := RefreshFeed(url)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Tried to fetch the feed, but some error occurred reading it")
	}
	feed.Title = html2text.HTML2Text(feed.Title)
	rssFeed := fmt.Sprintf("Fetched **%s** _(%s)_", feed.Title, html2text.HTML2Text(feed.Description))
	tag, err := dbPool.Exec(context.Background(), rssNewFeed, html2text.HTML2Text(feed.Title), url.String(), make(map[string]struct{}))
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Went to insert this feed into the database for later, and it didn't seem to like that." +
			" Maybe provide a less spicy feed? Or try some Pepto-Bismol")
	}
	log.Println(tag)
	return []string{rssFeed}, nil
}

// CommandList returns a list of aliases for the RSS Command
func (r RSS) CommandList() []string {
	return []string{"!rss"}
}

// Help returns the help message for the RSS Command
func (r RSS) Help() string {
	return "Subscribes to an RSS feed\n" +
		"- `!rss list` lists all subscribed RSS feeds\n" +
		"- `!rss find <id>` finds an RSS feed by it's numerical ID\n" +
		"- `!rss latest <id>` re-fetches the latest element of the feed and (if it hasn't already been posted) posts it"
}

// RSSInfo contains the posted information for a RSS feed item
type RSSInfo struct {
	Title       string
	Description string
}

// ReduceItem reduces a list of RSS items to a list of titles and secondary text (usually URLs)
func ReduceItem(items []*gofeed.Item) []RSSInfo {
	infoItems := []RSSInfo{}
	for _, item := range items {
		infoItems = append(infoItems, RSSInfo{
			Title: html2text.HTML2Text(item.Title),
			Description: func() string {
				secondary := item.Link
				if len(secondary) == 0 {
					if len(item.Enclosures) != 0 {
						return item.Enclosures[0].URL
					}
					return html2text.HTML2Text(item.Description)
				}
				return secondary
			}(),
		})
	}
	return infoItems
}

// RefreshFeed fetches a given RSS feed
func RefreshFeed(url *url.URL) (*gofeed.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	return fp.ParseURLWithContext(url.String(), ctx)
}

// SQL Helpers

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
