FROM golang:alpine AS linter
RUN apk add --no-cache --update git \
    && go get github.com/securego/gosec/cmd/gosec \
    && go get -u golang.org/x/lint/golint
WORKDIR /lint
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./pre-commit

FROM golang:alpine AS build
WORKDIR /build
# Caches all the dependency downloads
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"

FROM alpine:latest
# cowsay is in the testing repository so that needs to be added
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing/" >> /etc/apk/repositories \
    && apk add --no-cache --update cowsay fortune\
    && addgroup -g 1000 birbbot \
    && adduser -u 1000 -G birbbot -s /bin/sh -D birbbot
WORKDIR /home/birbbot
USER birbbot
COPY --from=build /build/birbbot .
CMD ./birbbot