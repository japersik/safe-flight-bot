package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient/avtmClient"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient/openstreetmap"
	"github.com/japersik/safe-flight-bot/internal/flyPlanner"
	"github.com/japersik/safe-flight-bot/internal/telegram"
	"log"
	"os"
)

func main() {
	tgBotToken := os.Getenv("TG_BOT_TOKEN")
	if tgBotToken == "" {
		log.Fatal("Empty env variable TG_BOT_TOKEN")
	}

	bot, err := tgbotapi.NewBotAPI(tgBotToken)
	if err != nil {
		return
	}

	avtm := avtmClient.NewAvtmClient()
	maps := openstreetmap.NewOpenStreetClient()

	flClient := flyDataClient.Client{WeatherInfoSource: avtm, ZoneInfoSource: avtm, LocalityInfoSource: maps}

	planner := flyPlanner.NewPlaner()

	myBot := telegram.NewBot(bot, flClient, planner)

	if err := myBot.Start(); err != nil {
		log.Fatal("Error bot starting: ", err)
	}
}
