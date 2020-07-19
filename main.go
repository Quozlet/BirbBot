package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quozlet.net/birbbot/app"
)

func main() {
	rand.Seed(time.Now().Unix())
	session, err := app.Start(os.Getenv("DISCORD_SECRET"))
	defer func() {
		// If a session is established, close it properly before exiting
		if session != nil {
			session.Close()
		}
	}()
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Println("Successful startup! Kill the process to stop the bot")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sig
}
