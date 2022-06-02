package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"github.com/japersik/safe-flight-bot/internal/flyPlanner"
	"github.com/japersik/safe-flight-bot/model"
)

type Bot struct {
	bot       *tgbotapi.BotAPI
	flyClient flyDataClient.Client
	planner   flyPlanner.Planner
}

func NewBot(bot *tgbotapi.BotAPI, client flyDataClient.Client, planner flyPlanner.Planner) *Bot {
	myBot := &Bot{bot: bot, flyClient: client, planner: planner}
	planner.SetNotifier(myBot)
	return myBot
}

func (b *Bot) Notify(flyPlan model.FlyPlan) error {
	text, err := b.getInfoText(flyPlan.Data.Coordinate, 100)
	if err != nil {
		return err
	}
	text = "test text from notify\n " + text
	msg := tgbotapi.NewMessage(flyPlan.Data.UserId, text)
	_, err = b.Send(msg)
	return err
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	//b.flyClient.CheckConditions(flyDataClient.Coordinate{
	//	Lng: 30,
	//	Lat: 60,
	//}, 2000)

	// Receiving and processing updates
	updates := b.bot.GetUpdatesChan(u)
	for update := range updates {
		go b.manageUpdate(update)
	}
	return nil
}

func (b Bot) manageUpdate(update tgbotapi.Update) {
	if update.CallbackData() != "" {
		b.handleCallback(update.FromChat(), update.CallbackData())
	}
	if update.Message == nil {
		// ignore any non-Message Updates
	} else if update.Message.Location != nil {
		// Обработка отправленной геолокации
		b.handleGeoLocationMessage(update.Message)
	} else if update.Message.IsCommand() {
		// Обработка отправленной команды
		b.handleCommand(update.Message)
	} else {
		// Обработка Остальных сообщений
		msg := tgbotapi.NewPoll(update.Message.Chat.ID, "123123", "123m", "123")
		//msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		b.bot.Send(msg)
	}
}

func (b *Bot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {

	return b.bot.Send(c)
}
