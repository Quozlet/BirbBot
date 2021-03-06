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

const (
	filterTableDefinition = "CREATE TABLE IF NOT EXISTS Filters " +
		"(ID SERIAL PRIMARY KEY, Regex TEXT UNIQUE NOT NULL, FeedID INTEGER REFERENCES Feeds(ID))"
	filterInsert = "INSERT INTO Filters(Regex) VALUES ($1) ON CONFLICT (Regex) DO NOTHING"
	filterApply  = "UPDATE Filters SET FeedID = $1 WHERE ID = $2"
	filterList   = "SELECT ID, Regex FROM Filters"
	filterSelect = "SELECT Regex FROM Filters WHERE FeedID = $1"
)

const (
	feedReadErrorMsg      = "Error occurred reading the feeds, aborting"
	invalidFeedIDErrorMsg = "%s is not a valid number to use as an ID"
)

// Filter is a command to store a regular expression (regex) for filtering RSS feeds.
type Filter struct{}

// Check creates the table if it does not already exists, with an error if unable to do so.
func (f Filter) Check(dbPool *pgxpool.Pool) error {
	tag, err := dbPool.Exec(context.Background(), filterTableDefinition)
	if err != nil {
		return err
	}

	log.Printf("Filter: %s", tag)

	return nil
}

// ProcessMessage for a Filter command will either apply or create a filter.
func (f Filter) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	splitContent := strings.Fields(m.Content)
	if len(splitContent) < 2 {
		return commands.NewError("There needs to be at least something to use as a subcommand or regex")
	}

	message := splitContent[1:]

	switch message[0] {
	case "list":
		return listRegex(response, m.ChannelID, dbPool)
	case "apply":
		return applyRegex(response, m.ChannelID, message[1:], dbPool)
	default:
		return handlePossibleRegex(response, m.ChannelID, m.Content, dbPool)
	}
}

func handlePossibleRegex(
	response chan<- commands.MessageResponse,
	channelID string,
	exp string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	possibleRegexIndex := strings.IndexRune(exp, ' ')

	if possibleRegexIndex < -1 {
		return commands.NewError(fmt.Sprintf("Failed to parse '%s'", exp))
	}

	regex, err := regexp.Compile(strings.TrimLeftFunc(string([]rune(exp)[possibleRegexIndex:]), unicode.IsSpace))

	if commandError = commands.CreateCommandError(
		fmt.Sprintf("Failed to parse '%s'", string([]rune(exp)[possibleRegexIndex:])),
		err,
	); commandError != nil {
		return commandError
	}

	tag, err := dbPool.Exec(context.Background(), filterInsert, regex.String())

	if commandError = commands.CreateCommandError(
		"Parsed as a valid regex, but failed to save. Try again!",
		err,
	); commandError != nil {
		return commandError
	}

	log.Printf("Filter: %s (actually inserted %s)", tag, regex)
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message:   "Saved successfully. Use `!filter apply` to apply for a feed",
	}

	return nil
}

func applyRegex(response chan<- commands.MessageResponse,
	channelID string,
	ids []string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	if len(ids) < 2 {
		return commands.NewError("Need to associate a given filter to given feed.\n\n" +
			"See `!help filter` for more info)")
	}

	regexID, err := strconv.ParseInt(ids[0], 0, 64)

	if commandError = commands.CreateCommandError(
		fmt.Sprintf(invalidFeedIDErrorMsg, ids[0]),
		err,
	); commandError != nil {
		return commandError
	}

	feedID, err := strconv.ParseInt(ids[1], 0, 64)

	if commandError = commands.CreateCommandError(
		fmt.Sprintf(invalidFeedIDErrorMsg, ids[1]),
		err,
	); commandError != nil {
		return commandError
	}

	tag, err := dbPool.Exec(context.Background(), filterApply, regexID, feedID)

	if commandError = commands.CreateCommandError(
		fmt.Sprintf("Failed to apply that filter. "+
			"Check that %d is a valid filter ID, and %d a valid RSS ID", regexID, feedID),
		err,
	); commandError != nil {
		return commandError
	}

	log.Printf("Filter: %s (actually inserted RegexID %d, FeedID %d)", tag, regexID, feedID)
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message:   fmt.Sprintf("Successfully associated %d to feed %d", regexID, feedID),
	}

	return nil
}

func listRegex(
	response chan<- commands.MessageResponse,
	channelID string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError

	rows, err := dbPool.Query(context.Background(), filterList)

	if commandError = commands.CreateCommandError(
		"Sorry, failed to lookup filters. Doesn't mean there aren't any though, so try again",
		err,
	); commandError != nil {
		return commandError
	}

	sentFilters := false

	for rows.Next() {
		var id int64

		var regex string
		if commandError = commands.CreateCommandError(
			feedReadErrorMsg,
			rows.Scan(&id, &regex),
		); commandError != nil {
			return commandError
		}

		sentFilters = true
		response <- commands.MessageResponse{
			ChannelID: channelID,
			Message:   fmt.Sprintf("%d: %s", id, regex),
		}
	}

	if commandError = commands.CreateCommandError(
		feedReadErrorMsg,
		rows.Err(),
	); commandError != nil {
		return commandError
	}

	if !sentFilters {
		response <- commands.MessageResponse{
			ChannelID: channelID,
			Message:   "No filters have been saved yet!",
		}

		return nil
	}

	return nil
}

// CommandList returns a list of aliases for the Filter Command.
func (f Filter) CommandList() []string {
	return []string{"filter"}
}

// Help returns the helper message for the Filter Command.
func (f Filter) Help() string {
	return "`filter <regular expression>` saves a regular expression filter to apply to a certain RSS feed.\n" +
		"_Check https://regex101.com/ to create and test regex._\n\n" +
		"- `filter apply <feed id> <regex id>` to apply the filter for an existing subscription.\n" +
		"_Each possible description will be filtered, and only the (first) matching for each item will be posted._\n\n" +
		"- `filter list` lists all set filters and their content"
}

// FetchRegex fetches the regex for a given RSS Feed's ID.
func FetchRegex(id int64, dbPool *pgxpool.Pool) *regexp.Regexp {
	var regexString string
	if err := dbPool.QueryRow(context.Background(), filterSelect, id).Scan(&regexString); err != nil {
		log.Println(err)

		return nil
	}
	// To be stored in the database it must've compiled.
	return regexp.MustCompile(regexString)
}
