FROM golang:1.15.6-alpine3.12 AS build
WORKDIR /build
COPY ["go.mod", "go.sum", "./"]
RUN apk add --no-cache --update git=2.26.2-r0 gcc=9.3.0-r2 libc-dev=0.7.2-r3 \
    && go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.34.1 \
    && go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./pre-commit \
    && GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"

FROM alpine:3.12.3
# cowsay is in the testing repository so that needs to be added
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing/" >> /etc/apk/repositories \
    && apk add --no-cache --update cowsay=3.04-r0 fortune=0.1-r1 ffmpeg=4.3.1-r0 tini=0.19.0-r0 \
    && addgroup -g 1000 birbbot \
    && adduser -u 1000 -G birbbot -s /bin/sh -D birbbot
ENTRYPOINT ["/sbin/tini", "--"]
WORKDIR /home/birbbot
USER birbbot
COPY --from=build /build/birbbot .
CMD ["./birbbot"]
