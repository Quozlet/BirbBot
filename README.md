# BirbBot

A Discord bot written in Go

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites

[Go](https://golang.org/) must be installed, and potentially [Docker](https://www.docker.com/).

For the `!fortune` and `!cowsay` commands, [`fortune`](https://www.ibiblio.org/pub/linux/games/amusements/fortune/!INDEX.html) and [`cowsay`](https://github.com/tnalpgge/rank-amateur-cowsay) should be installed

* macOS: [Homebrew](https://brew.sh/)
* Windows: [Scoop](https://scoop.sh/)

### Installing

To run in one step

```
DISCORD_SECRET=AbC_123! go run main.go
```

Alternatively, to build an executable binary and run

```
go build .
DISCORD_SECRET=AbC_123! ./birbbot
```

## Running the tests

`go test`

### And coding style tests

To format: `go fmt`

To [vet](): `go vet`

To lint: [`golint`](https://github.com/golang/lint)

## Deployment

To build with Docker

`docker build -t birbbot:<some version> .`

To run the Docker image (assuming a `.env` file exists in the form of the `.env.example` file)

`docker run --env-file .env birbbot:<some version>`

## Built With

* [DiscordGo](https://github.com/bwmarrin/discordgo) - Go bindings for Discord 
* [gofeed](https://github.com/mmcdole/gofeed) - Parse RSS and Atom feeds in Go 

## License

Licensed under the [Open Software License 3.0](https://spdx.org/licenses/OSL-3.0.html).


