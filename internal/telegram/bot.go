package telegram

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"github.com/japersik/safe-flight-bot/internal/flyPlanner"
	"github.com/japersik/safe-flight-bot/model"
	"strconv"
	"sync"
)

//Callback структура используется в tgbotapi.NewInlineKeyboardButtonData в сериализованном в json виде
type Callback struct {
	CallbackType CallbackType `json:"callbackType"`
	Data         interface{}  `json:"data"`
}

//CallbackType ...
type CallbackType int

//flyPlanStage используется для обозначения и хранения стадии создания автоматических уведомлений
type flyPlanStage int

const (
	dateTimeSelect flyPlanStage = iota
	notifications
)

//plannedFlightInfo используется для хранения информации о процессе создания автоматических уведомлений
type plannedFlightInfo struct {
	plan  model.FlyPlan
	stage flyPlanStage
}

type Bot struct {
	bot                 *tgbotapi.BotAPI
	flyClient           flyDataClient.Client
	planner             flyPlanner.Planner
	flightPlanningUsers map[int64]*plannedFlightInfo
	planMutex           *sync.Mutex
	markupsToDelete     map[int64]tgbotapi.Message
	markupsMutex        *sync.Mutex
}

func NewBot(bot *tgbotapi.BotAPI, client flyDataClient.Client, planner flyPlanner.Planner) *Bot {
	myBot := &Bot{
		bot:                 bot,
		flyClient:           client,
		planner:             planner,
		flightPlanningUsers: map[int64]*plannedFlightInfo{},
		planMutex:           &sync.Mutex{},
		markupsToDelete:     map[int64]tgbotapi.Message{},
		markupsMutex:        &sync.Mutex{},
	}
	return myBot
}

//Notify реализация интерфейса flyPlanner.Notifier
func (b *Bot) Notify(flyPlan model.FlyPlan) error {
	text, err := b.getInfoText(flyPlan.Data.Coordinate, 100)
	if err != nil {
		return err
	}
	text = "Автоматическое уведомление №" + strconv.FormatUint(flyPlan.FlyId, 10) + "\n" + text
	msg := tgbotapi.NewMessage(flyPlan.Data.UserId, text)
	cancelFlyNotifications, _ := json.Marshal(Callback{
		CallbackType: cancelFlyCallback,
		Data:         flyPlan.FlyId,
	})
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отключить уведомление", string(cancelFlyNotifications))),
	)
	msg.ReplyMarkup = numericKeyboard
	msg.ParseMode = "HTML"
	_, err = b.Send(msg)
	return err
}

//Start ..
func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)
	for update := range updates {
		go b.manageUpdate(update)
	}
	return nil
}

func (b *Bot) manageUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	//var chatId int64
	//if update.FromChat() != nil {
	//	chatId = update.FromChat().ID
	//} else if update.Message != nil {
	//	chatId = update.Message.Chat.ID
	//} else {
	//	chatId = update.Poll.
	//}
	b.markupsMutex.Lock()
	msg, ok := b.markupsToDelete[update.FromChat().ID]
	b.markupsMutex.Unlock()
	if ok {
		msg := tgbotapi.NewEditMessageText(update.FromChat().ID, msg.MessageID, msg.Text)
		b.Send(msg)
	}
	ok, _ = b.checkManageFlyPlanning(update)
	if ok {
		return
	}
	//fmt.Println("Not in plan mode")
	if update.CallbackData() != "" {
		b.handleCallback(update.FromChat(), update.CallbackQuery)
	} else if update.Message == nil {
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

type MyMessageConfig struct {
	tgbotapi.Chattable
	needToDeleteMarkup bool
}

func (b *Bot) SendWithMarkupToDelete(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return b.Send(MyMessageConfig{c, true})
}

func (b *Bot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	msg, err := b.bot.Send(c)
	if message, ok := c.(MyMessageConfig); ok && message.needToDeleteMarkup {
		b.markupsMutex.Lock()
		b.markupsToDelete[msg.Chat.ID] = msg
		b.markupsMutex.Unlock()
	}
	return msg, err
}
