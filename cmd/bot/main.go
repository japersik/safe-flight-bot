package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient/avtmClient"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient/openstreetmapClient"
	"github.com/japersik/safe-flight-bot/internal/flyPlanner"
	"github.com/japersik/safe-flight-bot/internal/telegram"
	"github.com/japersik/safe-flight-bot/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	// logger setup
	zLog := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.Lock(os.Stdout), zap.NewAtomicLevel()))
	logger.NewInstance(logger.NewZapLogger(zLog))
	logger.Info("program started")
	//signals setup
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	//tbBot setup
	tgBotToken := os.Getenv("TG_BOT_TOKEN")
	if tgBotToken == "" {
		logger.Fatal("Empty env variable TG_BOT_TOKEN")
	}
	bot, err := tgbotapi.NewBotAPI(tgBotToken)
	if err != nil {
		return
	}

	//
	avtm := avtmClient.NewAvtmClient()
	maps := openstreetmapClient.NewOpenStreetClient()
	flClient := flyDataClient.Client{WeatherInfoSource: avtm, ZoneInfoSource: avtm, LocalityInfoSource: maps}

	//mission planner setup
	planner := flyPlanner.NewPlaner("data/file.json")

	myBot := telegram.NewBot(bot, flClient, planner)
	planner.SetNotifier(myBot)
	planner.Init()
	planner.Start()
	go func() {
		<-shutdownSignal
		err := planner.SavePlans()
		if err != nil {
			logger.Error("exit the program, data file save error ", err)
		} else {
			logger.Error("exit the program, the data file is saved")
		}
		os.Exit(0)
	}()
	if err := myBot.Start(); err != nil {
		logger.Fatal("error bot starting: ", err)
	}
}
