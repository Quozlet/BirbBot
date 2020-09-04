FROM returntocorp/semgrep:f76e1a701bcdbb4a24a707e05fa0be83bf2000a7
WORKDIR /lint
COPY . .
RUN semgrep -f /lint/semgrep/go.yml /lint/

FROM golang:1.15.0-alpine3.12 AS build
WORKDIR /build
# Caches all the dependency downloads
COPY ["go.mod", "go.sum", "./"]
RUN apk add --no-cache --update git ffmpeg \
    && go get github.com/securego/gosec/cmd/gosec \
    && go get -u golang.org/x/lint/golint \
    && go get -u github.com/kisielk/errcheck \
    && go get honnef.co/go/tools/cmd/staticcheck \
    && go mod download
COPY . .
# Cache linting binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./pre-commit \
    && GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"

FROM alpine:3.12.0
# cowsay is in the testing repository so that needs to be added
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing/" >> /etc/apk/repositories \
    && apk add --no-cache --update cowsay fortune ffmpeg \
    && addgroup -g 1000 birbbot \
    && adduser -u 1000 -G birbbot -s /bin/sh -D birbbot
WORKDIR /home/birbbot
USER birbbot
COPY --from=build /build/birbbot .
CMD ./birbbot
