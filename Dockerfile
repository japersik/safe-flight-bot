FROM golang:1.18-alpine3.15 AS builder

COPY . /github.com/japersik/safe-flight-bot/
WORKDIR /github.com/japersik/safe-flight-bot/

RUN go mod download &&\
     go build -o ./bin/bot cmd/bot/main.go

FROM  alpine:3.15
RUN apk add --no-cache tzdata

WORKDIR /root/
COPY --from=0 /github.com/japersik/safe-flight-bot/bin/bot .

CMD ["./bot"]