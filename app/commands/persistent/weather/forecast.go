package weather

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
)

// Forecast is a Command to get the Forecast (today's, tomorrow's, or the day after's) for a given location
type Forecast struct{}

// Check validates the weather URL
func (f Forecast) Check(dbPool *pgxpool.Pool) error {
	return canFetchWeather(dbPool)
}

// ProcessMessage processes a given message and fetches the weather for the location specified for the day specified
func (f Forecast) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError
	message := strings.Fields(m.Content)[1:]
	// Start of extended forcast (lines 7-17)
	start, end := 7, 17
	url, err := createWeatherURL(message, m.Author.ID, dbPool)
	if commandError = commands.CreateCommandError(
		"Tried to create a plan to fetch the weather, but it failed",
		err,
	); commandError != nil {
		return commandError
	}
	if len(message) != 0 {
		log.Printf("Recognized variant %s, processing", message[0])
		switch strings.ToLower(message[0]) {
		case "tomorrow":
			start += weatherWidth
			end += weatherWidth
			url, err = createWeatherURL(message[1:], m.Author.ID, dbPool)
		case "last":
			start += 2 * weatherWidth
			end += 2 * weatherWidth
			url, err = createWeatherURL(message[1:], m.Author.ID, dbPool)
		}
	}
	if commandError = commands.CreateCommandError(
		"Failed to make a plan for getting the weather."+
			" Try again later (if this occurred when you thought a location was set, it probably isn't)",
		err,
	); commandError != nil {
		return commandError
	}
	forecast, err := detailedWeather(url, start, end)
	if commandError = commands.CreateCommandError(
		"Couldn't get the forecast for that location for some reason",
		err,
	); commandError != nil {
		return commandError
	}
	response <- commands.MessageResponse{
		ChannelID: m.ChannelID,
		Message:   forecast,
	}
	return nil
}

// CommandList returns a list of aliases for the Forecast Command
func (f Forecast) CommandList() []string {
	return []string{"forecast"}
}

// Help returns the help message for the Forecase Command
func (f Forecast) Help() string {
	return "Provides today's forecast for a location (either provided or set)\n" +
		"- `forecast tomorrow` gets the forecast for tomorrow\n" +
		"- `forecast last` gets the forecast for the day after next\n\n" +
		"To manage set locations, use the `w`/`weather set` or `w`/`weather clear` commands"
}
