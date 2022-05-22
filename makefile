.PHONY:

build:
	go build -o ./bin/bot cmd/bot/main.go

run: build
	./bin/bot

build-image:
	docker build -t japersik/save-flight-bot .

run-container:
	docker run --name tg-bot --env-file .env japersik/save-flight-bot

start-container:
	docker start tg-bot