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
	//fmt.Println("HERE")
	//fmt.Println(update)
	//var id int64
	//if update.FromChat() != nil {
	//	id = update.FromChat().ID
	//} else {
	//	id = update.Message.Chat.ID
	//}
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
				flyId, _ := b.planner.PlanFly(pFlightInfo.plan)
				//fmt.Println(flyId, err)
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
				pFlightInfo.plan.Notifications = []time.Duration{-48 * time.Hour, -24 * time.Hour, -12 * time.Hour, -3 * time.Hour,
					-2 * time.Hour, -1 * time.Hour, 0, 1 * time.Hour, 2 * time.Hour}
				flyId, _ := b.planner.PlanFly(pFlightInfo.plan)
				delete(b.flightPlanningUsers, update.FromChat().ID)
				return true, b.sendPlanCreated(update.FromChat(), flyId)
				//return true, b.sendSelectNotificationsTime(update.FromChat())
			}
			//case notifications:
			//	fmt.Println(pFlightInfo)
			//	fmt.Println(update.Message)
		}
	}
	return true, nil
}

//var selectTime = map[string]time.Duration{
//	"За 2 дня до запланированного времени":        -48 * time.Hour,
//	"За 1 день до запланированного времени":       -24 * time.Hour,
//	"За 12 часов до запланированного времени":     -12 * time.Hour,
//	"За 3 часа до запланированного времени":       -3 * time.Hour,
//	"За 2 часа до запланированного времени":       -2 * time.Hour,
//	"За 1 час до запланированного времени":        -time.Hour,
//	"В запланированное время":                     0,
//	"Через 1 час после запланированного времени":  time.Hour,
//	"Через 2 часа после запланированного времени": 2 * time.Hour,
//	"Через 3 часа после запланированного времени": 3 * time.Hour,
//}
//
//func (b *Bot) sendSelectNotificationsTime(chat *tgbotapi.Chat) error {
//	options := make([]string, 0, len(selectTime))
//	for i, _ := range selectTime {
//		options = append(options, i)
//	}
//	msg := tgbotapi.NewPoll(chat.ID, "В какое время стоит уведомить об обстановке?", options...)
//	msg.AllowsMultipleAnswers = true
//	_, err := b.bot.Send(msg)
//	return err
//}
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
		b.planMutex.Lock()
		delete(b.flightPlanningUsers, chat.ID)
		b.planMutex.Unlock()
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
