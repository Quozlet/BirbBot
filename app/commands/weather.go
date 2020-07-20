package commands

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type weatherReport struct {
	CurrentCondition []currentCondition `json:"current_condition"`
	Weather          []dailyWeather     `json:"weather"`
	NearestArea      []area             `json:"nearest_area"`
}

type currentCondition struct {
	FeelsLikeC     int           `json:"FeelsLikeC,string"`
	FeelsLikeF     int           `json:"FeelsLikeF,string"`
	Humidity       int           `json:"humidity,string"`
	WeatherDesc    []valueHolder `json:"weatherDesc"`
	Winddir16Point string        `json:"winddir16Point"`
	WindspeedKmph  int           `json:"windspeedKmph,string"`
	WindspeedMiles int           `json:"windspeedMiles,string"`
	TempC          int           `json:"temp_C,string"`
	TempF          int           `json:"temp_F,string"`
}

type valueHolder struct {
	Value string `json:"Value"`
}

type dailyWeather struct {
	MaxTempC int      `json:"maxtempC,string"`
	MaxTempF int      `json:"maxtempF,string"`
	MinTempC int      `json:"mintempC,string"`
	MinTempF int      `json:"mintempF,string"`
	Hourly   []hourly `json:"hourly"`
}

type hourly struct {
	ChanceOfRain int `json:"chanceofrain,string"`
	ChanceOfSnow int `json:"chanceofsnow,string"`
}

type area struct {
	AreaName []valueHolder `json:"areaName"`
	Country  []valueHolder `json:"country"`
	Region   []valueHolder `json:"region"`
}

const weatherURL = "https://wttr.in"

const weatherWidth = 10

// Weather is a Command to get the current weather for a location
type Weather struct{}

// Check validates the weather URL
func (w Weather) Check() error {
	_, err := url.Parse(weatherURL)
	if err != nil {
		return err
	}
	return loadWeatherDB()
}

// ProcessMessage processes a given message and fetches the weather for the location specified in the format specified
func (w Weather) ProcessMessage(m *discordgo.MessageCreate) (string, error) {
	splitCmd := strings.Fields(m.Content)
	if len(splitCmd) != 0 {
		if len(splitCmd) > 1 {
			switch strings.ToLower(splitCmd[1]) {
			case "simple":
				url, err := createWeatherURL(splitCmd[2:], m.Author.ID)
				if err != nil {
					return "", err
				}
				q := url.Query()
				q.Set("format", "4")
				url.RawQuery = q.Encode()
				body, err := weatherResponse(url)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%s: %s", strings.Title(strings.Join(splitCmd[2:], " ")), strings.Split(body, ":")[1]), nil
			case "classic":
				url, err := createWeatherURL(splitCmd[2:], m.Author.ID)
				if err != nil {
					return "", err
				}
				q := url.Query()
				q.Set("format", "j1")
				url.RawQuery = q.Encode()
				body, err := dataWeather(url)
				if err != nil {
					return "", err
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

			case "set":
				url, urlErr := createWeatherURL(splitCmd[2:], m.Author.ID)
				if urlErr != nil {
					return "", urlErr
				}
				if err := insertNewWeatherDB(m.Author.ID, url.String()); err != nil {
					return "", err
				}
				return "OK, saved your location", nil
			}
		}

		url, err := createWeatherURL(splitCmd[1:], m.Author.ID)
		if err != nil {
			return "", err
		}
		// Current forecast (lines 1-7)
		return detailedWeather(url, 1, 7)

	}
	return "", errors.New("Provide a location to get the weather for :)")
}

// CommandList returns a list of aliases for the Weather Command
func (w Weather) CommandList() []string {
	return []string{"!w", "!weather"}
}

// Help returns the help message for the Weather Command
func (w Weather) Help() string {
	return "Provides the current weather for a location " +
		"(use `!w`/`!weather simple` for a one line response, and `!w`/`!weather classic` for a detailed text response)\n" +
		"`!w`/`!weather set` will persist a default weather location if none is specified " +
		"(setting again will overwrite the previously set location)"
}

// Forecast is a Command to get the Forecast (today's, tomorrow's, or the day after's) for a given location
type Forecast struct{}

// Check validates the weather URL
func (f Forecast) Check() error {
	_, err := url.Parse(weatherURL)
	return err
}

// ProcessMessage processes a given message and fetches the weather for the location specified for the day specified
func (f Forecast) ProcessMessage(m *discordgo.MessageCreate) (string, error) {
	message := strings.Fields(m.Content)[1:]
	// Start of extended forcast (lines 7-17)
	start, end := 7, 17
	url, err := createWeatherURL(message, m.Author.ID)
	if err != nil {
		return "", err
	}
	if len(message) != 0 {
		switch strings.ToLower(message[0]) {
		case "tomorrow":
			start += weatherWidth
			end += weatherWidth
			url, err = createWeatherURL(message[1:], m.Author.ID)
		case "last":
			start += 2 * weatherWidth
			end += 2 * weatherWidth
			url, err = createWeatherURL(message[1:], m.Author.ID)
		}
	}
	if err != nil {
		return "", err
	}
	return detailedWeather(url, start, end)
}

// CommandList returns a list of aliases for the Forecast Command
func (f Forecast) CommandList() []string {
	return []string{"!forecast"}
}

// Help returns the help message for the Forecase Command
func (f Forecast) Help() string {
	return "Provides today's forecast for a location (either provided or set) " +
		"(use `!forecast tomorrow`/`!forecast last` to get tomorrow and the day after's forecast, respectively)\n\n" +
		"To set a preferred location, use the `!w`/`!weather set` command"
}

func createWeatherURL(location []string, authorID string) (*url.URL, error) {
	if len(location) == 0 {
		savedLocation, err := selectWeatherDB(authorID)
		if err != nil {
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

const subTableDefinition string = "CREATE TABLE IF NOT EXISTS Weather (User TEXT PRIMARY KEY, Location TEXT NOT NULL)"
const newSub string = "INSERT INTO Weather(User, Location) VALUES(?, ?) ON CONFLICT(User) DO UPDATE SET Location=excluded.Location"
const selectSub string = "SELECT Location FROM Weather WHERE User = ?"

func loadWeatherDB() error {
	db, err := sql.Open("sqlite3", "./birbbot.db")
	if err != nil {
		return err
	}
	_, err = db.Exec(subTableDefinition)
	if err != nil {
		return err
	}
	return nil
}

func insertNewWeatherDB(user string, location string) error {
	db, dbErr := sql.Open("sqlite3", "./birbbot.db")
	if dbErr != nil {
		return dbErr
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(err)
		}
	}()
	tx, txErr := db.Begin()
	if txErr != nil {
		return txErr
	}
	stmt, stmtErr := tx.Prepare(newSub)
	if stmtErr != nil {
		return stmtErr
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Println(err)
		}
	}()
	result, execErr := stmt.Exec(user, location)
	if execErr != nil {
		return execErr
	}
	lastInsertID, insertErr := result.LastInsertId()
	if insertErr != nil {
		log.Printf("Successfully inserted into DB, but an error (%s) occurred", insertErr)
	}
	rowsAffected, affectedErr := result.RowsAffected()
	if affectedErr != nil {
		log.Printf("Successfully inserted into DB, but an error (%s) occurred", affectedErr)
	}
	log.Printf("Inserted into database with insertion ID %d (%d rows affected)", lastInsertID, rowsAffected)
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func selectWeatherDB(user string) (string, error) {
	db, dbErr := sql.Open("sqlite3", "./birbbot.db")
	if dbErr != nil {
		return "", dbErr
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(err)
		}
	}()
	stmt, err := db.Prepare(selectSub)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Println(err)
		}
	}()
	var location string
	if err := stmt.QueryRow(user).Scan(&location); err != nil {
		return "", err
	}
	return location, nil
}
