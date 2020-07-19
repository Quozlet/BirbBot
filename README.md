# BirbBot

A Discord bot written in Go

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

[Go](https://golang.org/) or [Docker](https://www.docker.com/) must be installed.

#### Developing with Go

For the `!fortune` and `!cowsay` commands, [`fortune`](https://www.ibiblio.org/pub/linux/games/amusements/fortune/!INDEX.html) and [`cowsay`](https://github.com/tnalpgge/rank-amateur-cowsay) should be installed if developing locally with Go.

To install `fortune` or `cowsay`, use:
* macOS: [Homebrew](https://brew.sh/)
* Windows: [Scoop](https://scoop.sh/)

#### Developing with Docker

You must rebuild Docker each time, otherwise all dependencies are taken care of.

#### Bot Authorization

A Discord Bot token should be created to use the bot.
<!-- TODO: Create a mock environment to test the bot -->
Go to the [Discord Developer Portal](https://discordapp.com/developers/applications/), click "New Application", then go the "Bot" and click "Add Bot". The bot's token should be put into a file named `.env` in the format of [`.env.example`](.env.example). DO NOT have this token anywhere in the project when pushing to GitHub (.env is automatically [ignored](https://git-scm.com/docs/gitignore), see the [.gitignore](.gitignore)). If you accidentally include it somewhere, remove it and immediately regenerate a new one.

To test this bot with your server, copy the client ID for your application, and go to `https://discord.com/oauth2/authorize?client_id=<your client id>&scope=bot`.

### Running

Assuming you are in a terminal with the project as your current directory:

_(`.` means "current directory"), and could be substituted for an absolute path_

#### Go
To run in one step

```bash
DISCORD_SECRET=AbC_123! go run main.go
```

Alternatively, build an executable binary

```bash
go build .
```

Then run

```bash
DISCORD_SECRET=AbC_123! ./birbbot
```
#### Docker
To build with Docker

```bash
docker build -t birbbot:<some version> .
```

To run the Docker image (assuming a `.env` file exists in the form of the `.env.example` file)

```bash
docker run --env-file .env birbbot:<some version>
```

## Running the tests

`go test`

### And coding style tests

To format: `go fmt`

To vet: `go vet`

To lint: [`golint`](https://github.com/golang/lint)

## Built With

* [DiscordGo](https://github.com/bwmarrin/discordgo) - Go bindings for Discord
* [gofeed](https://github.com/mmcdole/gofeed) - Parse RSS and Atom feeds in Go

## License

Licensed under the [Open Software License 3.0](https://spdx.org/licenses/OSL-3.0.html).
