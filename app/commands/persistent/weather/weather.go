package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	handler "quozlet.net/birbbot/util"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"quozlet.net/birbbot/app/commands"
)

// Weather is a Command to get the current weather for a location
type Weather struct{}

// Check validates the weather URL
func (w Weather) Check(dbPool *pgxpool.Pool) error {
	return canFetchWeather(dbPool)
}

// ProcessMessage processes a given message and fetches the weather for the location specified in the format specified
func (w Weather) ProcessMessage(
	response chan<- commands.MessageResponse,
	m *discordgo.MessageCreate,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError
	splitCmd := strings.Fields(m.Content)
	if len(splitCmd) != 0 {
		log.Println("Valid weather command")
		if len(splitCmd) > 1 {
			log.Printf("Recognized variant %s", splitCmd[1])
			switch strings.ToLower(splitCmd[1]) {
			case "simple":
				return handleSimple(response, m.ChannelID, splitCmd[2:], m.Author.ID, dbPool)

			case "classic":
				return handleClassic(response, m.ChannelID, splitCmd[2:], m.Author.ID, dbPool)

			case "set":
				return setWeatherPreference(response, m.ChannelID, splitCmd[2:], m.Author.ID, dbPool)

			case "clear":
				return clearWeatherPreference(response, m.ChannelID, m.Author.ID, dbPool)

			}
		}

		url, err := createWeatherURL(splitCmd[1:], m.Author.ID, dbPool)
		if commandError = commands.CreateCommandError(
			"Tried to create plan to get weather, but it failed. "+
				"If this occurred when you thought a location was set, it probably isn't",
			err,
		); commandError != nil {
			return commandError
		}
		// Current forecast (lines 1-7)
		forecast, weatherErr := detailedWeather(url, 1, 7)
		if commandError = commands.CreateCommandError(
			"Unable to get the weather!"+
				" Sorry",
			weatherErr,
		); commandError != nil {
			return commandError
		}
		response <- commands.MessageResponse{
			ChannelID: m.ChannelID,
			Message:   forecast,
		}
		return nil

	}
	return commands.NewError("Provide a location to get the weather for :)")
}

// CommandList returns a list of aliases for the Weather Command
func (w Weather) CommandList() []string {
	return []string{"w", "weather"}
}

// Help returns the help message for the Weather Command
func (w Weather) Help() string {
	return "Provides the current weather for a location\n" +
		"_If a postal code is provided, put the country as a specifier: e.x. '12345 United States'_\n\n" +
		"- `w`/`weather simple` gives a one line weather update\n" +
		"- `w`/`weather classic` for a detailed text response\n" +
		"- `w`/`weather set` will persist a default weather location for the above commands (setting again will overwrite the previously set location)\n" +
		"- `w`/`weather clear` will clear your preferences (it will always return success unless a database error occurred)"
}

func handleClassic(
	response chan<- commands.MessageResponse,
	channelID string,
	location []string,
	discordUserID string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError
	url, err := createWeatherURL(location, discordUserID, dbPool)
	if commandError = commands.CreateCommandError(
		"Tried to create plan to get weather, but it failed.",
		err,
	); commandError != nil {
		return commandError
	}
	q := url.Query()
	q.Set("format", "j1")
	url.RawQuery = q.Encode()
	body, err := dataWeather(url)
	if commandError = commands.CreateCommandError(
		"Tried to get the weather forecast, but couldn't fetch it",
		err,
	); commandError != nil {
		return commandError
	}
	var precip string
	if body.Weather[0].Hourly[0].ChanceOfRain < body.Weather[0].Hourly[0].ChanceOfSnow {
		precip = fmt.Sprintf("%d%% chance of snow", body.Weather[0].Hourly[0].ChanceOfSnow)
	} else {
		precip = fmt.Sprintf("%d%% chance of rain", body.Weather[0].Hourly[0].ChanceOfRain)
	}
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message: fmt.Sprintf("%s, %dºF (%dºC) / feels like %dºF (%dºC) | High: %dºF (%dºC) | Low %dºF (%dºC) | Humidity: %d%% | Wind: %s @ %dmph (%dkm/h) | %s (%s, %s, %s)",
			body.CurrentCondition[0].WeatherDesc[0].Value,
			body.CurrentCondition[0].TempF,
			body.CurrentCondition[0].TempC,
			body.CurrentCondition[0].FeelsLikeF,
			body.CurrentCondition[0].FeelsLikeC,
			body.Weather[0].MaxTempF,
			body.Weather[0].MaxTempC,
			body.Weather[0].MinTempF,
			body.Weather[0].MinTempC,
			body.CurrentCondition[0].Humidity,
			body.CurrentCondition[0].Winddir16Point,
			body.CurrentCondition[0].WindspeedMiles,
			body.CurrentCondition[0].WindspeedKmph,
			precip,
			body.NearestArea[0].AreaName[0].Value,
			body.NearestArea[0].Region[0].Value,
			body.NearestArea[0].Country[0].Value),
	}
	return nil
}

func handleSimple(
	response chan<- commands.MessageResponse,
	channelID string,
	location []string,
	discordUserID string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError
	url, err := createWeatherURL(location, discordUserID, dbPool)
	if commandError = commands.CreateCommandError(
		"Tried to create plan to get weather, but it failed.",
		err,
	); commandError != nil {
		return commandError
	}
	q := url.Query()
	q.Set("format", "4")
	url.RawQuery = q.Encode()
	body, err := weatherResponse(url)
	if commandError = commands.CreateCommandError(
		"Tried to get the weather forecast, but couldn't fetch it",
		err,
	); commandError != nil {
		return commandError
	}
	if len(location) == 0 {
		response <- commands.MessageResponse{
			ChannelID: channelID,
			Message:   strings.Split(body, ":")[1],
		}
		return nil
	}
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message:   fmt.Sprintf("%s: %s", strings.Title(strings.Join(location, " ")), strings.Split(body, ":")[1]),
	}
	return nil
}

func setWeatherPreference(
	response chan<- commands.MessageResponse,
	channelID string,
	location []string,
	discordUserID string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError
	url, urlErr := createWeatherURL(location, discordUserID, dbPool)
	if commandError = commands.CreateCommandError(
		"Tried to create plan to get weather, but it failed.",
		urlErr,
	); commandError != nil {
		return commandError
	}
	tag, err := dbPool.Exec(context.Background(), weatherNewDefault, discordUserID, url.String())
	if commandError = commands.CreateCommandError(
		"Sorry, I couldn't save your location."+
			" An error occured",
		err,
	); commandError != nil {
		return commandError
	}
	log.Printf("Weather: %s (actually inserted %s for Discord user %s)", tag, weatherNewDefault, discordUserID)
	q := url.Query()
	q.Set("format", "j1")
	url.RawQuery = q.Encode()
	body, weatherLocationErr := dataWeather(url)
	if commandError = commands.CreateCommandError(
		"Your weather location was saved, but (FYI) I couldn't figure out the closes weather station."+
			" Double check it's a valid location",
		weatherLocationErr,
	); commandError != nil {
		return commandError
	}
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message: fmt.Sprintf("OK, saved your location."+
			" Closest weather station is %s, %s, %s",
			body.NearestArea[0].AreaName[0].Value,
			body.NearestArea[0].Region[0].Value,
			body.NearestArea[0].Country[0].Value),
	}
	return nil
}

func clearWeatherPreference(
	response chan<- commands.MessageResponse,
	channelID string,
	discordUserID string,
	dbPool *pgxpool.Pool,
) *commands.CommandError {
	var commandError *commands.CommandError
	tag, err := dbPool.Exec(context.Background(), weatherDrop, discordUserID)
	if commandError = commands.CreateCommandError(
		"Couldn't clear the database."+
			" A database error occured."+
			" Try again later or contact the server owner",
		err,
	); commandError != nil {
		return commandError
	}
	log.Printf("Weather: %s (actually remove default %s for a user)", tag, weatherDrop)
	response <- commands.MessageResponse{
		ChannelID: channelID,
		Message: fmt.Sprintf("Your preferences have been cleared from the database\n" +
			"_Due to automatic logging/backups, there may still be records of this information." +
			" To request their deletion please contact the owner of the server_"),
	}
	return nil
}

func createWeatherURL(location []string, authorID string, dbPool *pgxpool.Pool) (*url.URL, error) {
	if len(location) == 0 {
		var savedLocation string
		if err := dbPool.QueryRow(context.Background(), weatherSelect, authorID).Scan(&savedLocation); err != nil {
			return nil, err
		} else if len(savedLocation) == 0 {
			return nil, errors.New("provide (or set) a location to get the weather for :)")
		}
		savedLocationURL, err := url.Parse(savedLocation)
		if err != nil {
			return nil, err
		}
		return savedLocationURL, nil
	}
	url, err := url.Parse(weatherURL)
	if err != nil {
		return nil, err
	}
	url.Path = strings.Join(location, "+")
	q := url.Query()
	q.Set("no-terminal", "true")
	q.Set("narrow", "true")
	url.RawQuery = q.Encode()
	return url, nil
}

func weatherResponse(url *url.URL) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", err
	}
	// Force website to send back just text
	request.Header.Set("User-Agent", "curl")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(response.Body)
	defer handler.LogError(response.Body.Close())
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func detailedWeather(url *url.URL, startLine int, endLine int) (string, error) {
	body, err := weatherResponse(url)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("```\n%s```", strings.Join(strings.Split(body, "\n")[startLine:endLine], "\n")), nil
}

func dataWeather(url *url.URL) (*weatherReport, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	// Force website to send back just text
	request.Header.Set("User-Agent", "curl")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	report := weatherReport{}
	defer handler.LogError(response.Body.Close())
	if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
		return nil, err
	}
	return &report, nil
}
