package telegram

import (
	"encoding/json"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/model"
	"github.com/mitchellh/mapstructure"
	"time"
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
		coords := model.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}

		b.planner.PlanFly(model.FlyPlan{
			Data:          model.FlyData{Coordinate: coords, UserId: chat.ID},
			FlyDateTime:   time.Now(),
			Notifications: nil,
		})
		text = "Полет запланирован"
	case cancelFlyCallback:
		text = "Эта функция (отмены полётов) ещё не реализована, но скоро обязательно появится"
	case repeatRequestCallback:
		coords := model.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handleRepeatRequestCallback(chat, coords)
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	_, err = b.Send(msg)
	return err
}

func (b Bot) handleRepeatRequestCallback(chat *tgbotapi.Chat, coord model.Coordinate) error {
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
