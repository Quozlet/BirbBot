FROM golang:1.15.6-alpine3.12 AS build
WORKDIR /build
COPY ["go.mod", "go.sum", "./"]
RUN apk add --no-cache --update git=2.26.2-r0 gcc=9.3.0-r2 libc-dev=0.7.2-r3 \
    # CVE-2020-28928 avd.aquasec.com/nvd/cve-2020-28928
    musl=1.1.24-r10 musl-utils=1.1.24-r10 \
    && go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0 \
    && go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./pre-commit \
    && GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"

FROM alpine:3.12.1
# cowsay is in the testing repository so that needs to be added
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing/" >> /etc/apk/repositories \
    && apk add --no-cache --update cowsay=3.04-r0 fortune=0.1-r1 ffmpeg \
    # CVE-2020-28928 avd.aquasec.com/nvd/cve-2020-28928
    musl=1.1.24-r10 musl-utils=1.1.24-r10 \
    && addgroup -g 1000 birbbot \
    && adduser -u 1000 -G birbbot -s /bin/sh -D birbbot
WORKDIR /home/birbbot
USER birbbot
COPY --from=build /build/birbbot .
CMD ["./birbbot"]
