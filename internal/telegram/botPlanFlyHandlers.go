package telegram

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/model"
	"github.com/zsefvlol/timezonemapper"
	"time"
)

func (b Bot) checkManageFlyPlanning(update tgbotapi.Update) (bool, error) {
	b.planMutex.Lock()
	pFlightInfo, ok := b.flightPlanningUsers[update.FromChat().ID]
	b.planMutex.Unlock()
	if !ok {
		return false, nil
	}
	if update.CallbackData() != "" {
		b.handlePlanFlyCallbacks(update.FromChat(), update.CallbackQuery)
	} else if pFlightInfo.plan.IsEveryDayPlan {
		switch pFlightInfo.stage {

		case dateTimeSelect:
			if t, err := parseTime(update.Message.Text, pFlightInfo.plan.Data.Coordinate); err != nil {
				return true, b.sendFlyPlaningStatus(update.FromChat(), *pFlightInfo)
			} else {
				pFlightInfo.plan.FlyDateTime = t
				pFlightInfo.plan.Notifications = []time.Duration{0}
				flyId, err := b.planner.PlanFly(pFlightInfo.plan)
				fmt.Println(flyId, err)
				delete(b.flightPlanningUsers, update.FromChat().ID)
				return true, b.sendPlanCreated(update.FromChat(), flyId)
			}
		}
	} else {
		switch pFlightInfo.stage {
		case dateTimeSelect:
			if t, err := parseDateTime(update.Message.Text, pFlightInfo.plan.Data.Coordinate); err != nil {
				return true, err
			} else {
				pFlightInfo.plan.FlyDateTime = t
				pFlightInfo.plan.Notifications = []time.Duration{0}
				//b.planner.PlanFly(pFlightInfo.plan)
				pFlightInfo.stage = notifications
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Переход в стадию выбора оповещений")
				b.bot.Send(msg)
			}
		}
	}
	return true, nil
}

func (b *Bot) handlePlanFlyCallbacks(chat *tgbotapi.Chat, query *tgbotapi.CallbackQuery) error {
	var text string
	callbackData := query.Data
	callback := Callback{}
	err := json.Unmarshal([]byte(callbackData), &callback)
	if err != nil {
		return WrongCallbackErr
	}
	switch callback.CallbackType {
	case cancelPlanFlyCallback:
		return b.sendFlightPlanningCanceled(chat)
	default:
		fmt.Println(callback)
		text = " Еще не реализовано:("
	}
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	_, err = b.Send(msg)
	return err
}

func parseTime(str string, coordinate model.Coordinate) (time.Time, error) {
	layout := "15:04"
	timezone := timezonemapper.LatLngToTimezoneString(coordinate.Lat, coordinate.Lng)
	loc, _ := time.LoadLocation(timezone)
	return time.ParseInLocation(layout, str, loc)
}

func parseDateTime(str string, coordinate model.Coordinate) (time.Time, error) {
	layout := "02.01.2006 15:04"
	timezone := timezonemapper.LatLngToTimezoneString(coordinate.Lat, coordinate.Lng)
	loc, _ := time.LoadLocation(timezone)
	return time.ParseInLocation(layout, str, loc)
}

// Planner Handlers
func (b Bot) sendFlyPlaningStatus(chat *tgbotapi.Chat, status plannedFlightInfo) error {
	switch status.stage {
	case dateTimeSelect:
		if status.plan.IsEveryDayPlan {
			return b.sendNeedTimeSelect(chat)
		} else {
			return b.sendNeedDateSelect(chat)
		}
	default:
		return b.sendNotInPlanningMode(chat)
	}
}
func (b Bot) sendNotInPlanningMode(chat *tgbotapi.Chat) error {
	text :=
		`Вы не находитесь в режиме планирования`
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	_, err := b.Send(msg)
	return err
}
func (b Bot) sendNeedDateSelect(chat *tgbotapi.Chat) error {
	text :=
		`Вы находитесь в режиме планирования.
 Напишите дату и местное время в формате <b>01.12.2022 23:59</b> или нажмите кнопку "Отмена" для отмены планирования полета`
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
	_, err := b.SendWithMarkupToDelete(msg)
	return err
}
func (b Bot) sendNeedTimeSelect(chat *tgbotapi.Chat) error {
	text :=
		`Вы находитесь в режиме планирования.
 Напишите местное время в формате <b>23:59</b> или нажмите кнопку "Отмена" дял отмены планирования полета`
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
	_, err := b.SendWithMarkupToDelete(msg)
	return err
}
func (b Bot) sendPlanCreated(chat *tgbotapi.Chat, flyId uint64) error {
	text :=
		`Уведомление успешно запланировано.
Нажмите кнопку ниже для отмены`
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	cancelFlyNotifications, _ := json.Marshal(Callback{
		CallbackType: cancelFlyCallback,
		Data:         flyId,
	})
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отменить уведомление", string(cancelFlyNotifications))),
	)
	msg.ReplyMarkup = numericKeyboard
	_, err := b.Send(msg)
	return err
}

func (b Bot) sendFlightPlanningCanceled(chat *tgbotapi.Chat) error {
	text :=
		`Планирование полета/ежедневного уведомления отменено`
	msg := tgbotapi.NewMessage(chat.ID, text)
	msg.ParseMode = "HTML"
	_, err := b.Send(msg)
	return err
}
