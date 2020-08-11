package persistent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
)

const filterTableDefinition string = "CREATE TABLE IF NOT EXISTS Filters (ID SERIAL PRIMARY KEY, Regex TEXT UNIQUE NOT NULL, FeedID INTEGER REFERENCES Feeds(ID))"
const filterInsert string = "INSERT INTO Filters(Regex) VALUES ($1) ON CONFLICT (Regex) DO NOTHING"
const filterApply string = "UPDATE Filters SET FeedID = $1 WHERE ID = $2"
const filterList string = "SELECT ID, Regex FROM Filters"
const filterSelect string = "SELECT Regex FROM Filters WHERE FeedID = $1"

// Filter is a command to store a regular expression (regex) for filtering RSS feeds
type Filter struct{}

// Check creates the table if it does not already exists, with an error if unable to do so
func (f Filter) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), filterTableDefinition)
	if err != nil {
		return err
	}
	log.Printf("Filter: %s", tag)
	return nil
}

// ProcessMessage for a Filter command will either apply or create a filter
func (f Filter) ProcessMessage(m *discordgo.MessageCreate, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return nil, commands.NewError("There needs to be at least something to use as a subcommand or regex")
	}
	message := splitContent[1:]
	switch message[0] {
	case "list":
		return listRegex(dbPool)
	case "apply":
		return applyRegex(message[1:], dbPool)
	default:
		return handlePossibleRegex(m.Content, dbPool)
	}
}

func handlePossibleRegex(exp string, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	possibleRegexIndex := strings.IndexRune(exp, ' ')
	if possibleRegexIndex < -1 {
		return nil, commands.NewError(fmt.Sprintf("Failed to parse '%s'", exp))
	}
	regex, err := regexp.Compile(strings.TrimLeftFunc(string([]rune(exp)[possibleRegexIndex:]), unicode.IsSpace))
	if err != nil {
		log.Println(err)
		return nil, commands.NewError(fmt.Sprintf("Failed to parse '%s'", string([]rune(exp)[possibleRegexIndex:])))
	}
	tag, err := dbPool.Exec(context.Background(), filterInsert, regex.String())
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Parsed as a valid regex, but failed to save. Try again!")
	}
	log.Printf("Filter: %s (actually inserted %s)", tag, regex)
	return []string{"Saved successfully. Use `!filter apply` to apply for a feed"}, nil
}

func applyRegex(ids []string, dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	if len(ids) < 2 {
		return nil, commands.NewError("Need to associate a given filter to given feed.\n\n" +
			"See `!help filter` for more info)")
	}
	regexID, err := strconv.ParseInt(ids[0], 0, 64)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError(fmt.Sprintf("%s is not a valid number to use as an ID", ids[0]))
	}
	feedID, err := strconv.ParseInt(ids[1], 0, 64)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError(fmt.Sprintf("%s is not a valid number to use as an ID", ids[1]))
	}
	tag, err := dbPool.Exec(context.Background(), filterApply, regexID, feedID)
	if err != nil {
		return nil, commands.NewError(fmt.Sprintf("Failed to apply that filter. "+
			"Check that %d is a valid filter ID, and %d a valid RSS ID", regexID, feedID))
	}
	log.Printf("Filter: %s (actually inserted RegexID %d, FeedID %d)", tag, regexID, feedID)
	return []string{fmt.Sprintf("Successfully associated %d to feed %d", regexID, feedID)}, nil
}

func listRegex(dbPool *pgxpool.Pool) ([]string, *commands.CommandError) {
	rows, err := dbPool.Query(context.Background(), filterList)
	if err != nil {
		log.Println(err)
		return nil, commands.NewError("Sorry, failed to lookup filters. Doesn't mean there aren't any though, so try again")
	}
	filters := []string{}
	for rows.Next() {
		var id int64
		var regex string
		if err := rows.Scan(&id, &regex); err != nil {
			log.Println(err)
			return nil, commands.NewError("Error occurred reading the feeds, aborting")
		}
		filters = append(filters, fmt.Sprintf("%d: %s", id, regex))
	}
	if len(filters) == 0 {
		return []string{"No filters have been saved yet!"}, nil
	}
	return filters, nil
}

// CommandList returns a list of aliases for the Filter Command
func (f Filter) CommandList() []string {
	return []string{"!filter"}
}

// Help returns the helper message for the Filter Command
func (f Filter) Help() string {
	return "`!filter <regular expression>` saves a regular expression filter to apply to a certain RSS feed.\n" +
		"Check https://regex101.com/ to create and test regex.\n\n" +
		"Use `!filter apply <feed id> <regex id>` to apply the filter for an existing subscription. " +
		"Each possible description will be filtered, and only the (first) matching for each item will be posted.\n\n" +
		"`!filter list` lists all set filters and their content"
}

// FetchRegex fetches the regex for a given RSS Feed's ID
func FetchRegex(id int64, dbPool *pgxpool.Pool) *regexp.Regexp {
	var regexString string
	if err := dbPool.QueryRow(context.Background(), filterSelect, id).Scan(&regexString); err != nil {
		log.Println(err)
		return nil
	}
	// To be stored in the database it must've compiled
	return regexp.MustCompile(regexString)
}
