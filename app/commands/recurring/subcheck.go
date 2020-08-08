package recurring

import (
	"context"
	"crypto/sha512"
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
		var lastItem []byte
		if err := dbPool.QueryRow(context.Background(), persistent.RSSSelect, id).Scan(&title, &feedURL, &lastItem); err != nil {
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
		articleTitle, secondary := persistent.StringifyItem(feed.Items[0])
		sha := sha512.New()
		_, err = sha.Write([]byte(secondary))
		if err != nil {
			log.Println(err)
			return nil
		}
		hash := sha.Sum(nil)
		if fmt.Sprintf("%x", hash) != fmt.Sprintf("%x", lastItem) {
			tag, err := dbPool.Exec(context.Background(), persistent.RSSUpdateLastItem, hash, id)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println(tag)
			pendingMessages[channel] = append(pendingMessages[channel], fmt.Sprintf("%s\n**%s**\n%s", title, articleTitle, secondary))
		}
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil
	}
	return pendingMessages
}

// Frequency reports that subscriptions should be checked hourly
func (s SubCheck) Frequency() Frequency {
	return Hourly
}
