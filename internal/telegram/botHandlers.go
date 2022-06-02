package telegram

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/japersik/safe-flight-bot/model"
	"strings"
)

func (b *Bot) handleCommand(message *tgbotapi.Message) error {
	//fmt.Println(message.Command())
	switch message.Command() {
	case "start":
		return b.handleStartCommand(message)
	case "info":
		return b.handleInfoCommand(message)
	default:
		return b.handleUnknownCommand(message)
	}
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

func (b *Bot) handleUnknownCommand(message *tgbotapi.Message) error {
	text := "Эта команда не поддерживается"
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	_, err := b.Send(msg)
	return err
}

func (b *Bot) handleGeoLocationMessage(message *tgbotapi.Message) error {
	coord := model.Coordinate{
		Lng: message.Location.Longitude,
		Lat: message.Location.Latitude,
	}
	text, err := b.getInfoText(coord, 300)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
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
	//fmt.Println(b.flyClient.GetForecastWeather(coord))
	_, err = b.Send(msg)
	//fmt.Println(err)
	return err
}

func (b Bot) getInfoText(coord model.Coordinate, radius int) (string, error) {
	text := fmt.Sprintf("Полученые географические координаты:<b> %f ,%f </b>\n\n", coord.Lng, coord.Lat)
	locationInfo, err := b.flyClient.LocalityInfoSource.GetLocalityFlyInfo(coord)
	if err != nil {
		text += "К сожалению, не удалось получить информацию о населенных пунктах вблизи. \n\n"
	} else {
		text += "Точка находится в: " + locationInfo.Name + "\n"
		if locationInfo.FlyRestriction {
			text += "Полёты над населёнными пунктами <b>требуют согласования</b> с администрацией\n\n"
		}
	}

	zoneInfo, err := b.flyClient.CheckConditions(coord, radius)
	if err == nil {
		if zoneInfo.NearBoundaryZone {
			text += "Выбранные координаты находятся в 20-км приграничной зоне. Полёты здесь запрещены\n\n"
		} else {
			if len(zoneInfo.ActiveZones) == 0 {
				text += "В этом месте нет действующих зон ограничений полётов\n\n"
			} else {
				text += "В этом месте имеются зоны, <b>ограничивающие полеты</b>: " +
					strings.Join(zoneInfo.ActiveZones, ", ") + "\n\n"
			}
			if len(zoneInfo.InactiveZones) > 0 {
				text += "Также в данный момент <b>не действуют</b>, но могут стать активными следующие зоны:" +
					strings.Join(zoneInfo.InactiveZones, ", ") + "\n\n"
			}
		}
	} else {
		text += "К сожалению, не удалось получить информацию о зонах ограничения полетов от сервера. \n\n"
	}

	weatherInfo, err := b.flyClient.GetForecastWeather(coord)
	if err == nil {
		text += fmt.Sprintf("<b>Информация о погоде:</b> \n")
		text += fmt.Sprintf("Температура: %v *C\n", weatherInfo.Current.Temperature)
		text += fmt.Sprintf("Ветер: %v м/c, %v \n", weatherInfo.Current.WindSpeed, weatherInfo.Current.WindDeg)
		text += fmt.Sprintf("Облачность: %v%% \n", weatherInfo.Current.Humidity)
		text += fmt.Sprintf("Вероятность выпадения осадков: %v%% \n", weatherInfo.Current.PrecipProb*100)
		text += fmt.Sprintf("Видимость: %v м\n", weatherInfo.Current.Visibility)
		text += fmt.Sprintf("Давление: %v мм рт.ст. \n\n", weatherInfo.Current.Pressure)
	} else {
		text += "К сожалению, не удалось получить информацию о погоде от сервера. \n\n"
	}

	return text, nil
}
