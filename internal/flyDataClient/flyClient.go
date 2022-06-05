package flyDataClient

import "github.com/japersik/safe-flight-bot/model"

type WeatherInfoSource interface {
	GetForecastWeather(model.Coordinate) (*model.WeatherForecast, error)
	GetCurrentWeather(model.Coordinate) (*model.CurrentWeatherData, error)
}

type ZoneInfoSource interface {
	CheckConditions(model.Coordinate, int) (model.Condition, error)
}

type LocalityInfo struct {
	Name           string
	FlyRestriction bool
}

type LocalityInfoSource interface {
	GetLocalityFlyInfo(model.Coordinate) (*LocalityInfo, error)
}

type Client struct {
	WeatherInfoSource
	ZoneInfoSource
	LocalityInfoSource
}
