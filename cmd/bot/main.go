package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient/avtmClient"
	"github.com/japersik/safe-flight-bot/internal/telegram"
)

func main() {
	fmt.Println("Starting")
	bot, err := tgbotapi.NewBotAPI("tocken")
	if err != nil {
		return
	}
	avtm := avtmClient.NewAvmtClient()
	flClient := flyDataClient.Client{WeatherInfoSource: avtm, ZoneInfoSource: avtm}
	myBot := telegram.NewBot(bot, flClient)
	myBot.Start()
}
