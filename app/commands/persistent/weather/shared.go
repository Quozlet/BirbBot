package weather

import (
	"context"
	"log"
	"net/url"

	"github.com/jackc/pgx/v4/pgxpool"
)

const weatherTableDefinition string = "CREATE TABLE IF NOT EXISTS Weather (DiscordUserID TEXT PRIMARY KEY, Location TEXT NOT NULL)"
const weatherNewDefault string = "INSERT INTO Weather (DiscordUserID, Location) VALUES ($1, $2) ON CONFLICT(DiscordUserID) DO UPDATE SET Location=excluded.Location"
const weatherSelect string = "SELECT Location FROM Weather WHERE DiscordUserID = $1"
const weatherDrop string = "DELETE FROM WEATHER WHERE DiscordUserID = $1"

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

func canFetchWeather(dbPool *pgxpool.Pool) error {
	_, err := url.Parse(weatherURL)
	if err != nil {
		return err
	}
	tag, err := dbPool.Exec(context.Background(), weatherTableDefinition)
	if err != nil {
		return err
	}
	log.Printf("Weather/Forecast: %s", tag)
	return nil
}
