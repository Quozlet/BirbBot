package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const weatherURL = "https://wttr.in"

const weatherWidth = 10

// Weather is a Command to get the current weather for a location
type Weather struct{}

// Check validates the weather URL
func (w Weather) Check() error {
	_, err := url.Parse(weatherURL)
	return err
}

// ProcessMessage processes a given message and fetches the weather for the location specified in the format specified
func (w Weather) ProcessMessage(message ...string) (string, error) {
	if len(message) != 0 && message[0] == "simple" {
		url, err := createWeatherURL(message[1:])
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
		return fmt.Sprintf("%s: %s", strings.Title(strings.Join(message[1:], " ")), strings.Split(string(body), ":")[1]), nil
	}
	url, err := createWeatherURL(message)
	if err != nil {
		return "", err
	}
	// Current forecast (lines 1-7)
	return detailedWeather(url, 1, 7)

}

// CommandList returns a list of aliases for the Weather Command
func (w Weather) CommandList() []string {
	return []string{"!w", "!weather"}
}

// Help returns the help message for the Weather Command
func (w Weather) Help() string {
	return "Provides the current weather for a location (use `!w simple` for a one line response)"
}

// Forecast is a Command to get the Forecast (today's, tomorrow's, or the day after's) for a given location
type Forecast struct{}

// Check validates the weather URL
func (f Forecast) Check() error {
	_, err := url.Parse(weatherURL)
	return err
}

// ProcessMessage processes a given message and fetches the weather for the location specified for the day specified
func (f Forecast) ProcessMessage(message ...string) (string, error) {
	// Start of extended forcast (lines 7-17)
	start, end := 7, 17
	url, err := createWeatherURL(message)
	if err != nil {
		return "", err
	}
	if message[0] == "tomorrow" {
		start += weatherWidth
		end += weatherWidth
		url, err = createWeatherURL(message[1:])
	} else if message[0] == "last" {
		start += 2 * weatherWidth
		end += 2 * weatherWidth
		url, err = createWeatherURL(message[1:])
	}
	return detailedWeather(url, start, end)
}

// CommandList returns a list of aliases for the Forecast Command
func (f Forecast) CommandList() []string {
	return []string{"!forecast"}
}

// Help returns the help message for the Forecase Command
func (f Forecast) Help() string {
	return "Provides today's forecast for a location (use `!forecast tomorrow`/`!forecast last` to get tomorrow and the day after's forecast, respectively)"
}

func createWeatherURL(location []string) (*url.URL, error) {
	if len(location) == 0 {
		return nil, errors.New("Provide a location to get the weather for :)")
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
