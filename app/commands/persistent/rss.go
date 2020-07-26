package persistent

import (
	"context"
	"crypto/sha512"
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
const rssTableDefinition string = "CREATE TABLE IF NOT EXISTS Feeds (ID SERIAL PRIMARY KEY, Title TEXT NOT NULL, URL TEXT UNIQUE NOT NULL, LastItemHash BYTEA)"
const rssNewFeed string = "INSERT INTO Feeds(Title, URL, LastItemHash) VALUES ($1, $2, $3) ON CONFLICT (URL) DO NOTHING"
const rssList string = "SELECT ID, Title, URL, LastItemHash FROM Feeds"

// RSSSelect an RSS feed by ID
const RSSSelect string = "SELECT Title, URL, LastItemHash FROM Feeds WHERE ID = $1"

// RSSUpdateLastItem with a new hash
const RSSUpdateLastItem string = "UPDATE Feeds SET LastItemHash = $1 WHERE ID = $2"

// RSS is a command to fetch an RSS feed for validation
type RSS struct{}

// Check returns nil
func (r RSS) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), rssTableDefinition)
	if err != nil {
		return err
	}
	log.Printf("RSS: %s", tag)
	return nil
}

// ProcessMessage attempts to parse the first argument as a URL to an RSS feed, then fetch the first argument. If any step fails, an error is returned
func (r RSS) ProcessMessage(m *discordgo.MessageCreate, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return "", commands.NewError("Sure, let me test if that's valid.\n" +
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

func listFeeds(dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	feeds, err := selectAllFeedDB(dbPool)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Couldn't get a list of feeds from the database. " +
			"Try again later")
	}
	builder := strings.Builder{}
	for _, info := range feeds {
		builder.WriteString(fmt.Sprintf("ID: %d | %s\n", info.ID, info.Title))
	}
	if builder.Len() == 0 {
		return "", commands.NewError("Can't list, you haven't subscribed to any feeds yet")
	}
	return builder.String(), nil
}

func findFeedByID(args []string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	if len(args) == 1 {
		return "", commands.NewError("<insert 404 joke here> Look, you didn't provide anything to find")
	}
	id, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		log.Println(err)
		return "", commands.NewError(fmt.Sprintf(
			"Hey, so, uh, I need an _ID_, a number."+
				" %s is not a number", args[1]))
	}
	info, err := selectFeedDB(dbPool, id)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("I understood the ID, but the database says it's invalid." +
			" Can you double-check?")
	}
	return fmt.Sprintf("**%s** <%s>", info.Title, info.URL), nil
}

func fetchLatest(args []string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	if len(args) == 1 {
		return "", commands.NewError("**My Database**\nzilch\n\nTry providing an ID to search by")
	}
	id, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		log.Println(err)
		return "", commands.NewError(fmt.Sprintf(
			"Hey, so, uh, I need an _ID_, a number."+
				" %s is not a number", args[1]))
	}
	info, err := selectFeedDB(dbPool, id)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("I understood the ID, but the database says it's invalid. " +
			"Can you double-check?")
	}
	url, err := url.Parse(info.URL)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("This isn't good. Somehow an invalid feed URL was saved into the database for this ID")
	}
	feed, err := RefreshFeed(url)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Tried to fetch the feed, but some error occurred reading it")
	}
	if len(feed.Items) == 0 {
		return "", commands.NewError("Successfully fetched the feed, but it looks like it's empty")
	}
	latest := StringifyItem(feed.Items[0])
	sha := sha512.New()
	_, err = sha.Write([]byte(latest))
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Internal error." +
			" I'll probably be confused about the latest item in the feed until it's fetched again")
	}
	hash := sha.Sum(nil)
	if fmt.Sprintf("%x", hash) != fmt.Sprintf("%x", info.LastItem) {
		tag, err := dbPool.Exec(context.Background(), RSSUpdateLastItem, hash, id)
		if err != nil {
			log.Println(err)
			return "", commands.NewError("Internal error." +
				" I'll probably be confused about the latest item in the feed until it's fetched again")
		}
		log.Println(tag)
		return fmt.Sprintf("%s: %s", info.Title, latest), nil
	}
	return "Nothing new to report", nil
}

func storeNewFeed(userMsg string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	url, err := url.Parse(userMsg)
	if err != nil {
		log.Println(err)
		return "", commands.NewError(fmt.Sprintf("%s doesn't seem to be a valid URL", userMsg))
	}
	url.Scheme = "https"
	feed, err := RefreshFeed(url)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Tried to fetch the feed, but some error occurred reading it")
	}
	feed.Title = html2text.HTML2Text(feed.Title)
	rssFeed := fmt.Sprintf("Fetched **%s** _(%s)_", feed.Title, html2text.HTML2Text(feed.Description))
	tag, err := dbPool.Exec(context.Background(), rssNewFeed, html2text.HTML2Text(feed.Title), url.String(), nil)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Went to insert this feed into the database for later, and it didn't seem to like that." +
			" Maybe provide a less spicy feed? Or try some Pepto-Bismol")
	}
	log.Println(tag)
	return rssFeed, nil
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

// StringifyItem returns a string representation of an RSS Feed item
func StringifyItem(item *gofeed.Item) string {
	title := html2text.HTML2Text(item.Title)
	secondary := item.Link
	if len(secondary) == 0 {
		if len(item.Enclosures) != 0 {
			secondary = item.Enclosures[0].URL
		} else {
			secondary = html2text.HTML2Text(item.Description)
		}
	}
	return fmt.Sprintf("**%s**\n%s", title, secondary)
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
	ID       int64
	Title    string
	URL      string
	LastItem []byte
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
		var lastItem []byte
		if err := rows.Scan(&id, &title, &url, &lastItem); err != nil {
			return nil, err
		}
		info = append(info, &feedInfo{
			ID:       id,
			Title:    title,
			URL:      url,
			LastItem: lastItem,
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
	var lastItem []byte
	if err := dbPool.QueryRow(context.Background(), RSSSelect, id).Scan(&title, &url, &lastItem); err != nil {
		return nil, err
	}
	return &feedInfo{ID: id, Title: title, URL: url, LastItem: lastItem}, nil
}
