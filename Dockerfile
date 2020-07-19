FROM golang:alpine
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing/" >> /etc/apk/repositories \
    && apk update \
    && apk add --no-cache fortune cowsay
RUN addgroup -g 1000 birbbot \
    && adduser -u 1000 -G birbbot -s /bin/sh -D birbbot
WORKDIR /home/birbbot
USER birbbot
COPY . .
RUN go build .
CMD ./birbbot