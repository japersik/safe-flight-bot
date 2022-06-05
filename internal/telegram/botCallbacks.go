package telegram

import (
	"encoding/json"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/model"
	"github.com/mitchellh/mapstructure"
)

type Callback struct {
	CallbackType CallbackType `json:"callbackType"`
	Data         interface{}  `json:"data"`
}

type CallbackType int

const (
	planFlyCallback CallbackType = iota
	planFlyEdNotifications
	cancelPlanFlyCallback
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
		return b.handlePlanFlyCallback(chat, coords)
	case planFlyEdNotifications:
		coords := model.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handlePlanFlyEdNotifications(chat, coords)
	case cancelPlanFlyCallback:
		text = "Планирование полета отменено"
	case cancelFlyCallback:
		text = "Эта функция (отмены полётов) ещё не реализована, но скоро обязательно появится"
	case repeatRequestCallback:
		coords := model.Coordinate{}
		err := mapstructure.Decode(callback.Data, &coords)
		if err != nil {
			return WrongCallbackErr
		}
		return b.handleRepeatRequestCallback(chat, coords)
	default:
		text = " Еще не реализовано:("
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
	b.flightPlanningUsers[chat.ID] = plannedFlightInfo{
		plan: model.FlyPlan{
			Data:           model.FlyData{Coordinate: coord, UserId: chat.ID},
			IsEveryDayPlan: true,
		},
		stage: timeSelect,
	}
	return b.handleTimeSelect(chat)
}

func (b Bot) handlePlanFlyCallback(chat *tgbotapi.Chat, coord model.Coordinate) error {
	b.planMutex.Lock()
	defer b.planMutex.Unlock()
	b.flightPlanningUsers[chat.ID] = plannedFlightInfo{
		plan: model.FlyPlan{
			Data:           model.FlyData{Coordinate: coord, UserId: chat.ID},
			IsEveryDayPlan: false,
		},
		stage: dateSelect,
	}
	return b.handleDateSelect(chat)
}

// Planner Handlers
func (b Bot) handleDateSelect(chat *tgbotapi.Chat) error {
	text :=
		`Вы находитесь в режиме планирования.
 Напишите дату в формате дд.мм.гггг или нажмите кнопку "Отмена" дял отмены планирования полета`
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	callbackPlanEveryDayNotifications, _ := json.Marshal(Callback{
		CallbackType: cancelPlanFlyCallback,
		Data:         struct{}{},
	})
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", string(callbackPlanEveryDayNotifications))),
	)
	msg.ReplyMarkup = numericKeyboard
	_, err := b.Send(msg)
	return err
}
func (b Bot) handleTimeSelect(chat *tgbotapi.Chat) error {
	text :=
		`Вы находитесь в режиме планирования.
 Напишите время в формате 29:59 или нажмите кнопку "Отмена" дял отмены планирования полета`
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	callbackPlanEveryDayNotifications, _ := json.Marshal(Callback{
		CallbackType: cancelPlanFlyCallback,
		Data:         struct{}{},
	})
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", string(callbackPlanEveryDayNotifications))),
	)
	msg.ReplyMarkup = numericKeyboard
	_, err := b.Send(msg)
	return err
}
