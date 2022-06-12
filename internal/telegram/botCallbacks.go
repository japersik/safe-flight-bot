package telegram

import (
	"encoding/json"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/model"
	"github.com/mitchellh/mapstructure"
)

const (
	planFlyCallback CallbackType = iota
	planFlyEdNotifications
	cancelFlyCallback
	repeatRequestCallback
	cancelPlanFlyCallback
)

var (
	WrongCallbackErr = errors.New("wrong Callback")
)

func (b *Bot) handleCallback(chat *tgbotapi.Chat, query *tgbotapi.CallbackQuery) error {
	var text string
	callbackData := query.Data

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
		return b.handlePlanFlyCallback(chat, coords)
	case planFlyEdNotifications:
		coords := model.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handlePlanFlyEdNotifications(chat, coords)
	case cancelFlyCallback:
		query.Message.ReplyMarkup = nil
		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, query.Message.Text)
		b.Send(edit)
		var flyId uint64
		err := mapstructure.Decode(callback.Data, &flyId)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handleCancelFlyCallback(chat, flyId)
	case repeatRequestCallback:
		coords := model.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handleRepeatRequestCallback(chat, coords)
	default:
		text = "Еще не реализовано:("
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	_, err = b.Send(msg)
	return err
}

func (b Bot) handleRepeatRequestCallback(chat *tgbotapi.Chat, coord model.Coordinate) error {
	text, err := b.getInfoText(coord, 100)
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
func (b Bot) handlePlanFlyEdNotifications(chat *tgbotapi.Chat, coord model.Coordinate) error {
	b.planMutex.Lock()
	defer b.planMutex.Unlock()
	b.flightPlanningUsers[chat.ID] = &plannedFlightInfo{
		plan: model.FlyPlan{
			Data:           model.FlyData{Coordinate: coord, UserId: chat.ID},
			IsEveryDayPlan: true,
		},
		stage: dateTimeSelect,
	}
	return b.sendNeedTimeSelect(chat)
}

func (b Bot) handlePlanFlyCallback(chat *tgbotapi.Chat, coord model.Coordinate) error {
	b.planMutex.Lock()
	defer b.planMutex.Unlock()
	b.flightPlanningUsers[chat.ID] = &plannedFlightInfo{
		plan: model.FlyPlan{
			Data:           model.FlyData{Coordinate: coord, UserId: chat.ID},
			IsEveryDayPlan: false,
		},
		stage: dateTimeSelect,
	}
	return b.sendNeedDateSelect(chat)
}

func (b Bot) handleCancelFlyCallback(chat *tgbotapi.Chat, id uint64) error {
	err := b.planner.CancelFly(id)
	msg := tgbotapi.NewMessage(chat.ID, "Уведомление успешно отключено")
	if err != nil {
		msg = tgbotapi.NewMessage(chat.ID, "Это уведомление уже было отключено")
	}
	msg.ParseMode = "HTML"
	_, err = b.Send(msg)
	return err

}
