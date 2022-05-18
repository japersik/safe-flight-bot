package telegram

import (
	"encoding/json"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"github.com/mitchellh/mapstructure"
)

type Callback struct {
	CallbackType CallbackType `json:"callbackType"`
	Data         interface{}  `json:"data"`
}

type CallbackType int

const (
	planFlyCallback CallbackType = iota
	cancelFlyCallback
	repeatRequestCallback
)

var (
	WrongCallbackErr = errors.New("wrong Callback")
)

func (b *Bot) handleCallback(chat *tgbotapi.Chat, callbackData string) error {
	var text string

	//fmt.Println(callbackData)
	callback := Callback{}
	err := json.Unmarshal([]byte(callbackData), &callback)
	if err != nil {
		return WrongCallbackErr
	}
	switch callback.CallbackType {
	case planFlyCallback:
		text = "Эта функция (планирования полётов) ещё не реализована, но скоро обязательно появится"
	case cancelFlyCallback:
		text = "Эта функция (отмены полётов) ещё не реализована, но скоро обязательно появится"
	case repeatRequestCallback:
		coords := flyDataClient.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handleRepeatRequestCallback(chat, coords)
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	_, err = b.Send(msg)
	return err
}

func (b Bot) handleRepeatRequestCallback(chat *tgbotapi.Chat, coord flyDataClient.Coordinate) error {
	text, err := b.getInfoText(coord, 300)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"

	if _, err = b.Send(msg); err != nil {
		return err
	}
	return nil
}
