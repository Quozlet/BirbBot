package recurring

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"

	handler "quozlet.net/birbbot/util"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands/persistent"
)

// SubCheck routines checks for new items in the feed.
type SubCheck struct{}

const subList string = "SELECT FeedID, Channel FROM Subscriptions"

var errNoFeedItems = errors.New("Fetched ok, but no items in feed")

// Check will look for updates in subscribed feeds.
func (s SubCheck) Check(dbPool *pgxpool.Pool) map[string][]string {
	rows, err := dbPool.Query(context.Background(), subList)
	if err != nil {
		log.Println(err)

		return nil
	}

	pendingMessages := make(map[string][]string)
	// For each ID, map channel to pending messages, return chunk
	for rows.Next() {
		if err := processSubCheckRow(dbPool, &rows, &pendingMessages); err != nil {
			continue
		}
	}

	if err := rows.Err(); err != nil {
		log.Println(err)

		return nil
	}

	return pendingMessages
}

// Frequency reports that subscriptions should be checked hourly.
func (s SubCheck) Frequency() Frequency {
	return HalfHourly
}

// SubCleanup runs once a day to cleanup the growing list of posted content.
type SubCleanup struct{}

// Check will load the current elements of the feed and insert them as posted.
func (s SubCleanup) Check(dbPool *pgxpool.Pool) map[string][]string {
	rows, err := dbPool.Query(context.Background(), subList)
	if err != nil {
		log.Println(err)

		return nil
	}
	// For each ID, map channel to pending messages, return chunk
	for rows.Next() {
		if err := processSubCleanupRow(dbPool, &rows); err != nil {
			log.Println(err)

			continue
		}
	}

	if err := rows.Err(); err != nil {
		log.Println(err)

		return nil
	}

	return nil
}

// Frequency is the frequency of the sub cleanup.
func (s SubCleanup) Frequency() Frequency {
	return Daily
}

func processSubCheckRow(dbPool *pgxpool.Pool, rows *pgx.Rows, pendingMessages *map[string][]string) error {
	var id int64

	var channel string
	if err := (*rows).Scan(&id, &channel); err != nil {
		return err
	}

	var title string

	var feedURL string

	var lastItems map[string]struct{}
	if err := dbPool.QueryRow(
		context.Background(),
		persistent.RSSSelect,
		id,
	).Scan(&title,
		&feedURL,
		&lastItems,
	); err != nil {
		return err
	}

	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return err
	}

	feed, err := persistent.RefreshFeed(parsedURL)
	if err != nil {
		return err
	}

	if len(feed.Items) == 0 {
		return errNoFeedItems
	}

	urls := findUniqueURLs(
		persistent.ReduceItem(
			feed.Items,
			persistent.FetchRegex(id, dbPool),
		),
		lastItems,
		title,
		pendingMessages,
		channel,
	)

	tag, err := dbPool.Exec(context.Background(), persistent.RSSUpdateLastItem, func() map[string]struct{} {
		for desc := range lastItems {
			(*urls)[desc] = struct{}{}
		}

		return *urls
	}(), id)
	handler.LogError(err)
	log.Printf("SubCheck: %s (for ID %d)", tag, id)

	return nil
}

func processSubCleanupRow(dbPool *pgxpool.Pool, rows *pgx.Rows) error {
	var id int64

	var channel string
	if err := (*rows).Scan(&id, &channel); err != nil {
		return err
	}

	var title string

	var feedURL string

	var lastItems map[string]struct{}
	if err := dbPool.QueryRow(context.Background(),
		persistent.RSSSelect,
		id,
	).Scan(&title,
		&feedURL,
		&lastItems,
	); err != nil {
		return err
	}

	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return err
	}

	feed, err := persistent.RefreshFeed(parsedURL)
	if err != nil {
		return err
	}

	items := persistent.ReduceItem(feed.Items, persistent.FetchRegex(id, dbPool))
	urls := make(map[string]struct{})

	for _, item := range items {
		urls[item.Description] = struct{}{}
	}

	if _, err := dbPool.Exec(context.Background(), persistent.RSSUpdateLastItem, urls, id); err != nil {
		return err
	}

	log.Printf("SubCheck: inserted %d items for %d", len(urls), id)

	return nil
}

func findUniqueURLs(
	items []persistent.RSSInfo,
	lastItems map[string]struct{},
	title string,
	pendingMessages *map[string][]string,
	channel string,
) *map[string]struct{} {
	urls := make(map[string]struct{})

	for _, item := range items {
		_, contained := lastItems[item.Description]
		if !contained {
			(*pendingMessages)[channel] = append((*pendingMessages)[channel],
				fmt.Sprintf("%s\n**%s**\n%s",
					title,
					item.Title,
					item.Description,
				))
			urls[item.Description] = struct{}{}
		}
	}

	log.Printf("Identified %d new items, and %d existing ones", len(urls), len(lastItems))

	return &urls
}
