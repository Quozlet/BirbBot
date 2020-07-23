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
	session, err := app.Start(os.Getenv("DISCORD_SECRET"), dbPool)
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
		log.Fatal(err)
		return
	}

	log.Println("Successful startup! Kill the process to stop the bot")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sig
}
