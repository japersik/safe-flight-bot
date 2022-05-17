package telegram

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
)

func (b *Bot) handleCommand(message *tgbotapi.Message) error {
	fmt.Println(message.Command())
	switch message.Command() {
	case "start":
		return b.handleStartCommand(message)
	case "info":
		return b.handleInfoCommand(message)
	default:
	}
	return nil
}

func (b *Bot) handleStartCommand(message *tgbotapi.Message) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Привет, отправь мне геолокацию для получения информации о возможности полётов")
	numericKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonLocation("Проверить мой местоположение"),
		),
	)
	msg.ReplyMarkup = numericKeyboard
	_, err := b.Send(msg)
	return err
}

func (b *Bot) handleInfoCommand(message *tgbotapi.Message) error {
	text := fmt.Sprintf("Бот %s (@%s) - <b>не</b>официальный бот для работы с сервисом https://map.avtm.center \n"+
		"Здесь можно узнать информацию об ограничениях полётов, погоде и "+
		"запланировать полётную миссию с уведомлением о погоде и ограничениях. ", b.bot.Self.FirstName, b.bot.Self.UserName)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "HTML"
	numericKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonLocation("Проверить моё местоположение"),
		),
	)
	msg.ReplyMarkup = numericKeyboard
	_, err := b.Send(msg)
	return err
}

func (b *Bot) handleGeoLocationCommand(message *tgbotapi.Message) error {
	coord := flyDataClient.Coordinate{
		Lng: message.Location.Longitude,
		Lat: message.Location.Latitude,
	}
	helloText := fmt.Sprintf("Выбранные географические координаты: %f ,%f \n"+
		"Полёт дрона здесь разрешен/запрещен/требует согласования\n"+
		"Боллее подробная информация: [ссылка на сайт]\n"+
		"Информация о погоде сайчас:", coord.Lng, coord.Lat)

	msg := tgbotapi.NewMessage(message.Chat.ID, helloText)
	msg.ParseMode = "HTML"

	callbackPlanFly, _ := json.Marshal(Callback{
		CallbackType: planFlyCallback,
		Data:         coord,
	})
	callbackRepeatRequest, _ := json.Marshal(Callback{
		CallbackType: repeatRequestCallback,
		Data:         coord,
	})
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Запланировать полёт тут ", string(callbackPlanFly)),
		tgbotapi.NewInlineKeyboardButtonData("Повторить запрос ", string(callbackRepeatRequest)),
	),
	)

	msg.ReplyMarkup = numericKeyboard
	fmt.Println(b.flyClient.GetForecastWeather(coord))
	_, err := b.Send(msg)
	fmt.Println(err)
	return err
}
