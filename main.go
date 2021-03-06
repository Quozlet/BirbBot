package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quozlet.net/birbbot/app"

	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	rand.Seed(time.Now().Unix())

	dbPool, dbErr := pgxpool.Connect(context.Background(), fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("DATABASE_USER"),
		url.QueryEscape(os.Getenv("DATABASE_PASSWORD")),
		os.Getenv("DATABASE_URL"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_NAME")))
	if dbErr != nil {
		log.Fatal(dbErr)

		return
	}
	defer dbPool.Close()

	ticker := app.Timers{
		Daily:           time.NewTicker(time.Hour * 24),
		Hourly:          time.NewTicker(time.Hour),
		Minutely:        time.NewTicker(time.Minute),
		FiveMinutely:    time.NewTicker(time.Minute * 5),
		TenMinutely:     time.NewTicker(time.Minute * 10),
		QuarterHourly:   time.NewTicker(time.Minute * 15),
		HalfHourly:      time.NewTicker(time.Minute * 30),
		QuarterToHourly: time.NewTicker(time.Minute * 45),
	}

	defer ticker.StopAll()
	session, err := app.Start(os.Getenv("DISCORD_SECRET"), dbPool, &ticker)

	defer func() {
		// If a session is established, close it properly before exiting
		if session != nil {
			if sessionErr := session.Close(); sessionErr != nil {
				log.Fatal(sessionErr)

				return
			}
		}
	}()

	if err != nil {
		log.Println(err)

		return
	}

	log.Println("Completed startup! Kill the process to stop the bot")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sig
}
