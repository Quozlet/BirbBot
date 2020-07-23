package commands

import (
	"context"
	"crypto/sha512"
	"errors"
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
)

// RSS is a command to fetch an RSS feed for validation
type RSS struct{}

// Check returns nil
func (r RSS) Check(dbPool *pgxpool.Pool) error {
	return loadFeedDB(dbPool)
}

// ProcessMessage attempts to parse the first argument as a URL to an RSS feed, then fetch the first argument. If any step fails, an error is returned
func (r RSS) ProcessMessage(m *discordgo.MessageCreate, dbPool *pgxpool.Pool) (string, error) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return "", errors.New("Sure, let me test if that's valid.\n" +
			"Here comes the feed: _You are a horrible person_. " +
			"I'm serious, that's what's in the feed: _\"A horrible person\"_. We weren't even testing for that")
	}
	message := strings.Fields(strings.ToLower(m.Content))[1:]
	switch message[0] {
	case "list":
		return handleList(dbPool)

	case "find":
		return feedByID(message, dbPool)
	case "latest":
		return handleLatest(message, dbPool)
	default:
		url, err := url.Parse(message[0])
		if err != nil {
			return "", err
		}
		url.Scheme = "https"
		feed, err := refreshFeed(url)
		if err != nil {
			return "", err
		}
		feed.Title = html2text.HTML2Text(feed.Title)
		rssFeed := fmt.Sprintf("Fetched **%s** _(%s)_", feed.Title, html2text.HTML2Text(feed.Description))
		if err := insertNewFeedDB(dbPool, feed, url); err != nil {
			return "", err
		}
		return rssFeed, nil
	}
}

func handleList(dbPool *pgxpool.Pool) (string, error) {
	feeds, err := selectAllFeedDB(dbPool)
	if err != nil {
		return "", err
	}
	builder := strings.Builder{}
	for _, info := range feeds {
		builder.WriteString(fmt.Sprintf("ID: %d | %s\n", info.ID, info.Title))
	}
	if builder.Len() == 0 {
		return "", errors.New("Can't list, you haven't subscribed to any feeds yet")
	}
	return builder.String(), nil
}

func feedByID(args []string, dbPool *pgxpool.Pool) (string, error) {
	if len(args) == 1 {
		return "", errors.New("<insert 404 joke here> Look, you didn't provide anything to find")
	}
	id, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		return "", err
	}
	info, err := selectFeedDB(dbPool, id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("**%s** <%s>", info.Title, info.URL), nil
}

func handleLatest(args []string, dbPool *pgxpool.Pool) (string, error) {
	if len(args) == 1 {
		return "", errors.New("**My Database**\nzilch\n\nTry providing an ID to search by")
	}
	id, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		return "", err
	}
	info, err := selectFeedDB(dbPool, id)
	if err != nil {
		return "", err
	}
	url, err := url.Parse(info.URL)
	if err != nil {
		return "", err
	}
	feed, err := refreshFeed(url)
	if err != nil {
		return "", err
	}
	if len(feed.Items) == 0 {
		return "", errors.New("Successfully fetched the feed, but it looks like it's empty")
	}
	latest := stringifyItem(feed.Items[0])
	sha := sha512.New()
	_, err = sha.Write([]byte(latest))
	if err != nil {
		return "", err
	}
	hash := sha.Sum(nil)
	if fmt.Sprintf("%x", hash) != fmt.Sprintf("%x", info.LastItem) {
		if err := updateLatestFeedDB(dbPool, hash, id); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s: %s", info.Title, latest), nil
	}
	return "Nothing new to report", nil
}

// CommandList returns a list of aliases for the RSS Command
func (r RSS) CommandList() []string {
	return []string{"!sub", "!rss"}
}

// Help returns the help message for the RSS Command
func (r RSS) Help() string {
	return "Subscribes to an RSS feed\n" +
		"- `!rss`/`!sub list` lists all subscribed RSS feeds\n" +
		"- `!rss`/`!sub find <id>` finds an RSS feed by it's numerical ID\n" +
		"- `!rss`/`!sub latest <id>` re-fetches the latest element of the feed and (if it hasn't already been posted) posts it"
}

func stringifyItem(item *gofeed.Item) string {
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

func refreshFeed(url *url.URL) (*gofeed.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	return fp.ParseURLWithContext(url.String(), ctx)
}

// SQL Definitions/Helpers

const feedTableDefinition string = "CREATE TABLE IF NOT EXISTS Feeds (ID SERIAL PRIMARY KEY, Title TEXT NOT NULL, URL TEXT UNIQUE NOT NULL, LastItemHash BYTEA)"
const feedNew string = "INSERT INTO Feeds(Title, URL, LastItemHash) VALUES ($1, $2, $3) ON CONFLICT (URL) DO NOTHING"
const feedList string = "SELECT ID, Title, URL, LastItemHash FROM Feeds"
const feedSelect string = "SELECT Title, URL, LastItemHash FROM Feeds WHERE ID = $1"
const feedUpdate string = "UPDATE Feeds SET LastItemHash = $1 WHERE ID = $2"

type feedInfo struct {
	ID       int64
	Title    string
	URL      string
	LastItem []byte
}

func loadFeedDB(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), feedTableDefinition)
	if err != nil {
		return err
	}
	log.Println(tag)
	return nil
}

func insertNewFeedDB(dbPool *pgxpool.Pool, feed *gofeed.Feed, url *url.URL) error {
	tag, err := dbPool.Exec(context.Background(), feedNew, html2text.HTML2Text(feed.Title), url.String(), nil)
	if err != nil {
		return err
	}
	log.Println(tag)
	return nil
}

func selectAllFeedDB(dbPool *pgxpool.Pool) ([]*feedInfo, error) {
	rows, err := dbPool.Query(context.Background(), feedList)
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
	if err := dbPool.QueryRow(context.Background(), feedSelect, id).Scan(&title, &url, &lastItem); err != nil {
		return nil, err
	}
	return &feedInfo{ID: id, Title: title, URL: url, LastItem: lastItem}, nil
}

func updateLatestFeedDB(dbPool *pgxpool.Pool, hash []byte, id int64) error {
	tag, err := dbPool.Exec(context.Background(), feedUpdate, hash, id)
	if err != nil {
		return err
	}
	log.Println(tag)
	return nil
}
