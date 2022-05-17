package telegram

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func (b *Bot) handleCallback(chat *tgbotapi.Chat, callbackData string) error {
	var text string
	callback := Callback{}
	err := json.Unmarshal([]byte(callbackData), &callback)
	if err != nil {
		fmt.Println("Wrong Callback")
		return err
	}
	switch callback.CallbackType {
	case planFlyCallback:
		text = "Эта функция (планирования полётов) ещё не реализована, не скоро обязательно появится"
	case cancelFlyCallback:
		text = "Эта функция (отмены полётов) ещё не реализована, не скоро обязательно появится"
	case repeatRequestCallback:
		text = "Эта функция (повторного запроса информации) ещё не реализована, не скоро обязательно появится"
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	_, err = b.Send(msg)
	return err
}
