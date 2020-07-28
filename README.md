# BirbBot

A Discord bot written in Go

## Getting Started

These instructions will get the project up and running on your local machine for development and testing purposes.

### Prerequisites

[Go](https://golang.org/) or [Docker](https://www.docker.com/) must be installed.

A [package manager](https://en.wikipedia.org/wiki/Package_manager) is recommended for these or any optional dependencies below
* macOS: [Homebrew](https://brew.sh/)
* Windows: [Scoop](https://scoop.sh/)

Access to a [PostgreSQL](https://www.postgresql.org/) database is required.

#### Developing with Go

For the `!fortune` and `!cowsay` commands, [`fortune`](https://www.ibiblio.org/pub/linux/games/amusements/fortune/!INDEX.html) and [`cowsay`](https://github.com/tnalpgge/rank-amateur-cowsay) should be installed if developing locally with Go.

Persistent data (such as `!sub`/`!rss` subscriptions or saved `!w`/`!weather` locations) are stored in a PostgreSQL database. This requires some PostgreSQL database to be accessible to the application at startup, either from the local network (installation or VM/container), or remotely.

_The chosen PostgresSQL Go library ([pgx](https://github.com/jackc/pgx)) can perform certain optimizations if it's the only database, thus the lack of a fallback database if no PostgreSQL instance can be accessed._

Consider copying [`pre-commit`](pre-commit) as a [Git hook](https://git-scm.com/docs/githooks): `cp pre-commit .git/hooks/pre-commit`.

#### Developing with Docker

You must rebuild the Docker container for any change, but all dependencies will be resolved inside the container.

A [Docker compose](https://docs.docker.com/compose/) script is provided that instantiates a clean PostgreSQL instance to be used with the containerized application. `docker-compose build` creates the container, and `docker-compose up` starts them. The bot will (by design) crash until it can acquire a database connection.

#### Bot Authorization

A Discord Bot token should be created to use the bot.
<!-- TODO: Create a mock environment to test the bot -->
Go to the [Discord Developer Portal](https://discordapp.com/developers/applications/), click "New Application", then go the "Bot" and click "Add Bot". The bot's token should be put into a file named `.env` in the format of [`.env.example`](.env.example). DO NOT have this token anywhere in the project when pushing to GitHub (.env is automatically [ignored](https://git-scm.com/docs/gitignore), see the [.gitignore](.gitignore)). If you accidentally include it somewhere, remove it and immediately regenerate a new one.

To test this bot with your server, copy the client ID for your application, and go to `https://discord.com/oauth2/authorize?client_id=<your client id>&scope=bot`.

### Running

Assuming you are in a terminal with the project as your current directory:

_(`.` means "current directory"), and could be substituted for an absolute path_

#### Go
To run in one step (provided the required [environmental variables](https://en.wikipedia.org/wiki/Environment_variable#Assignment) are set):
```bash
go run main.go
```

Alternatively, build an executable binary with `go build`.
Then run:

```bash
./birbbot
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

Alternatively, `docker-compose` will take care of additional dependencies (like PostgreSQL).
- `docker-compose build`: Build or rebuild services
- `docker-compose up`: Create and start containers
- `docker-compose down`: Stop and remove containers

## Transferring data

The simplest way to transfer data is to make a backup of the database.

**Assuming the Docker compose script set up the database:**

Dump the whole database from a container:
`docker exec <postgres container name> pg_dumpall -c -U <database username> > backup.sql`

To restore the database on a machine:
`cat backup.sql | docker exec -i <postgres container name> psql -U <database username>`

## Running the tests

`go test`

### And coding style tests

To format: `go fmt`

To vet: `go vet`

To lint: [`golint`](https://github.com/golang/lint)

## License

Licensed under the [Open Software License 3.0](https://spdx.org/licenses/OSL-3.0.html).
