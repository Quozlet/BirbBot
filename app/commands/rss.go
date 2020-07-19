package commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
)

// TODO: Save feeds in database

// RSS is a command to fetch an RSS feed for validation
type RSS struct{}

// Check returns nil
func (r RSS) Check() error {
	// TODO: Check DB connection
	return nil
}

// ProcessMessage attempts to parse the first argument as a URL to an RSS feed, then fetch the first argument. If any step fails, an error is returned
func (r RSS) ProcessMessage(message ...string) (string, error) {
	if len(message) == 0 {
		return "", errors.New("Sure, let me test if that's valid.\n" +
			"Here comes the feed: _You are a horrible person_. " +
			"I'm serious, that's what's in the feed: _\"A horrible person\"_. We weren't even testing for that")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	url, err := url.Parse(message[0])
	if err != nil {
		return "", err
	}
	url.Scheme = "https"
	feed, err := fp.ParseURLWithContext(url.String(), ctx)
	if err != nil {
		return "", err
	}
	if feed == nil {
		return "", fmt.Errorf("Could not fetch an RSS feed for %s", url.String())
	}
	rssFeed := fmt.Sprintf("Fetched **%s** (%s)", html2text.HTML2Text(feed.Title), html2text.HTML2Text(feed.Description))
	if len(feed.Items) != 0 {
		firstItem := feed.Items[0]
		secondary := firstItem.Link
		if len(secondary) == 0 {
			if len(firstItem.Enclosures) != 0 {
				secondary = firstItem.Enclosures[0].URL
			} else {
				secondary = html2text.HTML2Text(firstItem.Description)
			}
		}
		rssFeed = fmt.Sprintf("%s\n\nLatest item\n**%s**\n%s", rssFeed, html2text.HTML2Text(firstItem.Title), secondary)
	}
	return rssFeed, nil
}

// CommandList returns a list of aliases for the RSS Command
func (r RSS) CommandList() []string {
	return []string{"!sub", "!rss"}
}

// Help returns the help message for the RSS Command
func (r RSS) Help() string {
	return "Provides some information about an RSS feed (title, description, first element)"
}
