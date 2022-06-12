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
	"os/signal"
	"syscall"
)

func main() {
	log.SetOutput(os.Stdout)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

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

	planner := flyPlanner.NewPlaner("data/file.json")

	myBot := telegram.NewBot(bot, flClient, planner)
	planner.SetNotifier(myBot)
	planner.Init()
	planner.Start()
	go func() {
		<-shutdownSignal
		err := planner.SavePlans()
		if err != nil {
			log.Print("exit the program, data file save error ", err)
		} else {
			log.Println("exit the program, the data file is saved.")
		}

		os.Exit(0)
	}()
	if err := myBot.Start(); err != nil {
		log.Fatal("error bot starting: ", err)
	}
}
