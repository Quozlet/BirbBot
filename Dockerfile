FROM golang:alpine
# cowsay is in the testing repository so that needs to be added
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing/" >> /etc/apk/repositories \
    && apk update \
    && apk add --no-cache fortune cowsay
# Create a user to run the app
RUN addgroup -g 1000 birbbot \
    && adduser -u 1000 -G birbbot -s /bin/sh -D birbbot
WORKDIR /home/birbbot
USER birbbot
# Caches all the dependency downloads
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
RUN go build .
CMD ./birbbot