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
func (w Weather) ProcessMessage(m *discordgo.MessageCreate, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	splitCmd := strings.Fields(m.Content)
	if len(splitCmd) != 0 {
		log.Println("Valid weather command")
		if len(splitCmd) > 1 {
			log.Printf("Recognized variant %s", splitCmd[1])
			switch strings.ToLower(splitCmd[1]) {
			case "simple":
				return handleSimple(splitCmd[2:], m.Author.ID, dbPool)

			case "classic":
				return handleClassic(splitCmd[2:], m.Author.ID, dbPool)

			case "set":
				return setWeatherPreference(splitCmd[2:], m.Author.ID, dbPool)

			case "clear":
				return clearWeatherPreference(m.Author.ID, dbPool)

			}
		}

		url, err := createWeatherURL(splitCmd[1:], m.Author.ID, dbPool)
		if err != nil {
			log.Println(err)
			return "", commands.NewError("Tried to create plan to get weather, but it failed. " +
				"If this occurred when you thought a location was set, it probably isn't")
		}
		// Current forecast (lines 1-7)
		forecast, weatherErr := detailedWeather(url, 1, 7)
		if weatherErr != nil {
			log.Println(err)
			return "", commands.NewError("Unable to get the weather!" +
				" Sorry")
		}
		return forecast, nil

	}
	return "", commands.NewError("Provide a location to get the weather for :)")
}

// CommandList returns a list of aliases for the Weather Command
func (w Weather) CommandList() []string {
	return []string{"!w", "!weather"}
}

// Help returns the help message for the Weather Command
func (w Weather) Help() string {
	return "Provides the current weather for a location\n" +
		"_If a postal code is provided, put the country as a specifier: e.x. '12345 United States'_\n\n" +
		"(use `!w`/`!weather simple` for a one line response, and `!w`/`!weather classic` for a detailed text response)\n" +
		"`!w`/`!weather set` will persist a default weather location if none is specified " +
		"(setting again will overwrite the previously set location)\n" +
		"`!w`/`!weather clear` will clear your preferences (it will always return success unless a database error occurred)"
}

func handleClassic(location []string, discordUserID string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	url, err := createWeatherURL(location, discordUserID, dbPool)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Tried to create plan to get weather, but it failed.")
	}
	q := url.Query()
	q.Set("format", "j1")
	url.RawQuery = q.Encode()
	body, err := dataWeather(url)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Tried to get the weather forecast, but couldn't fetch it")
	}
	var precip string
	if body.Weather[0].Hourly[0].ChanceOfRain < body.Weather[0].Hourly[0].ChanceOfSnow {
		precip = fmt.Sprintf("%d%% chance of snow", body.Weather[0].Hourly[0].ChanceOfSnow)
	} else {
		precip = fmt.Sprintf("%d%% chance of rain", body.Weather[0].Hourly[0].ChanceOfRain)
	}
	return fmt.Sprintf("%s, %dºF (%dºC) / feels like %dºF (%dºC) | High: %dºF (%dºC) | Low %dºF (%dºC) | Humidity: %d%% | Wind: %s @ %dmph (%dkm/h) | %s (%s, %s, %s)",
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
		body.NearestArea[0].Country[0].Value), nil
}

func handleSimple(location []string, discordUserID string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	url, err := createWeatherURL(location, discordUserID, dbPool)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Tried to create plan to get weather, but it failed.")
	}
	q := url.Query()
	q.Set("format", "4")
	url.RawQuery = q.Encode()
	body, err := weatherResponse(url)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Tried to get the weather forecast, but couldn't fetch it")
	}
	if len(location) == 0 {
		return strings.Split(body, ":")[1], nil
	}
	return fmt.Sprintf("%s: %s", strings.Title(strings.Join(location, " ")), strings.Split(body, ":")[1]), nil
}

func setWeatherPreference(location []string, discordUserID string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	url, urlErr := createWeatherURL(location, discordUserID, dbPool)
	if urlErr != nil {
		log.Println(urlErr)
		return "", commands.NewError("Tried to create plan to get weather, but it failed.")
	}
	tag, err := dbPool.Exec(context.Background(), weatherNewDefault, discordUserID, url.String())
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Sorry, I couldn't save your location." +
			" An error occured")
	}
	log.Println(tag)
	q := url.Query()
	q.Set("format", "j1")
	url.RawQuery = q.Encode()
	body, weatherLocationErr := dataWeather(url)
	if weatherLocationErr != nil {
		log.Println(weatherLocationErr)
		return "", commands.NewError("Your weather location was saved, but (FYI) I couldn't figure out the closes weather station." +
			" Double check it's a valid location")
	}
	return fmt.Sprintf("OK, saved your location."+
			" Closest weather station is %s, %s, %s",
			body.NearestArea[0].AreaName[0].Value,
			body.NearestArea[0].Region[0].Value,
			body.NearestArea[0].Country[0].Value),
		nil
}

func clearWeatherPreference(discordUserID string, dbPool *pgxpool.Pool) (string, *commands.CommandError) {
	tag, err := dbPool.Exec(context.Background(), weatherDrop, discordUserID)
	if err != nil {
		log.Println(err)
		return "", commands.NewError("Couldn't clear the database." +
			" A database error occured." +
			" Try again later or contact the server owner")
	}
	log.Println(tag)
	return fmt.Sprintf("Your preferences have been cleared from the database\n" +
		"_Due to automatic logging/backups, there may still be records of this information." +
		" To request their deletion please contact the owner of the server_"), nil
}

func createWeatherURL(location []string, authorID string, dbPool *pgxpool.Pool) (*url.URL, error) {
	if len(location) == 0 {
		var savedLocation string
		if err := dbPool.QueryRow(context.Background(), weatherSelect, authorID).Scan(&savedLocation); err != nil {
			return nil, err
		} else if len(savedLocation) == 0 {
			return nil, errors.New("Provide (or set) a location to get the weather for :)")
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
	defer response.Body.Close()
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
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
		return nil, err
	}
	return &report, nil
}
