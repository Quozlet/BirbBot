package recurring

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands/persistent"
)

// SubCheck routines checks for new items in the feed
type SubCheck struct{}

const subList string = "SELECT FeedID, Channel FROM Subscriptions"

// Check will look for updates in subscribed feeds
func (s SubCheck) Check(dbPool *pgxpool.Pool) map[string][]string {
	rows, err := dbPool.Query(context.Background(), subList)
	if err != nil {
		log.Println(err)
		return nil
	}
	pendingMessages := map[string][]string{}
	// For each ID, map channel to pending messages, return chunk
	for rows.Next() {
		var id int64
		var channel string
		if err := rows.Scan(&id, &channel); err != nil {
			log.Println(err)
			continue
		}
		var title string
		var feedURL string
		var lastItems map[string]struct{}
		if err := dbPool.QueryRow(context.Background(), persistent.RSSSelect, id).Scan(&title, &feedURL, &lastItems); err != nil {
			log.Println(err)
			continue
		}
		parsedURL, err := url.Parse(feedURL)
		if err != nil {
			log.Println(err)
			continue
		}
		feed, err := persistent.RefreshFeed(parsedURL)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(feed.Items) == 0 {
			log.Printf("Fetched %s ok, but no items in feed", feedURL)
			continue
		}
		items := persistent.ReduceItem(feed.Items, persistent.FetchRegex(id, dbPool))
		urls := make(map[string]struct{})
		for _, item := range items {
			_, contained := lastItems[item.Description]
			if !contained {
				pendingMessages[channel] = append(pendingMessages[channel], fmt.Sprintf("%s\n**%s**\n%s", title, item.Title, item.Description))
			}
			urls[item.Description] = struct{}{}
		}
		tag, err := dbPool.Exec(context.Background(), persistent.RSSUpdateLastItem, urls, id)
		if err != nil {
			log.Println(err)
		}
		log.Printf("SubCheck: %s (actually inserted %d items for %d)", tag, len(urls), id)
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil
	}
	return pendingMessages
}

// Frequency reports that subscriptions should be checked hourly
func (s SubCheck) Frequency() Frequency {
	return HalfHourly
}
