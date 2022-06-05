package model

import "time"

type FlyPlan struct {
	Data           FlyData         `json:"data"`
	FlyId          uint64          `json:"flyId"`
	FlyDateTime    time.Time       `json:"flyDateTime"`
	Notifications  []time.Duration `json:"notifications"`
	IsEveryDayPlan bool            `json:"isEveryDayPlan"`
}
type FlyData struct {
	Coordinate Coordinate `json:"coordinate"`
	Radius     int        `json:"radius"`
	UserId     int64      `json:"userId"`
}

type Coordinate struct {
	Lng float64 `json:"lng"`
	Lat float64 `json:"lat"`
}
type WeatherForecast struct {
	Current WeatherData   `json:"current"`
	Hourly  []WeatherData `json:"hourly"`
}

type WeatherData struct {
	PrecipProb  float64
	Temperature float64
	WindSpeed   float64
	WindDeg     float64
	Pressure    int
	Humidity    int
	Visibility  int
	Clouds      int
	Timestamp   time.Time
}

type CurrentWeatherData struct {
	Description string
	Temperature float64
	WindSpeed   float64
	WindDeg     float64
	Pressure    int
	Humidity    int
	Visibility  int
}

type Condition struct {
	DaylightHours       bool     `json:"daylightHours"`
	HasIntersections    bool     `json:"hasIntersections"`
	IntoCountryBoundary bool     `json:"intoCountryBoundary"`
	NearBoundaryZone    bool     `json:"nearBoundaryZone"`
	Permanent           bool     `json:"permanent"`
	PolarDayOrNight     bool     `json:"polarDayOrNight"`
	LocalTimeInLocation string   `json:"localTimeInLocation"`
	Sunrise             string   `json:"sunrise"`
	Sunset              string   `json:"sunset"`
	ActiveZones         []string `json:"activeZones"`
	InactiveZones       []string `json:"InactiveZones"`
}
