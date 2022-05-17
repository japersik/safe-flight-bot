package flyDataClient

import "time"

type Coordinate struct {
	Lng float64 `json:"lng"`
	Lat float64 `json:"lat"`
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
	DaylightHours       bool                `json:"daylightHours"`
	HasIntersections    bool                `json:"hasIntersections"`
	IntoCountryBoundary bool                `json:"intoCountryBoundary"`
	NearBoundaryZone    bool                `json:"nearBoundaryZone"`
	Permanent           bool                `json:"permanent"`
	PolarDayOrNight     bool                `json:"polarDayOrNight"`
	LocalTimeInLocation string              `json:"localTimeInLocation"`
	Sunrise             string              `json:"sunrise"`
	Sunset              string              `json:"sunset"`
	Zones               map[string]ZoneInfo `json:"map"`
}

type ZoneInfo struct {
	InactiveCodes      []interface{} `json:"inactive"`
	ActiveCodes        []interface{} `json:"active"`
	IntersectionCodes  []string      `json:"intersectionCodes"`
	CompletedWithError bool          `json:"completedWithError"`
	FullTime           int           `json:"fullTime"`
	ComputeTime        int           `json:"computeTime"`
	SelectTime         int           `json:"selectTime"`
}

type WeatherInfoSource interface {
	GetForecastWeather(Coordinate) (*WeatherData, error)
	GetCurrentWeather(Coordinate) (*CurrentWeatherData, error)
}

type ZoneInfoSource interface {
	CheckConditions(Coordinate, int) (int, error)
}

type LocalityInfo struct {
	Name           string
	FlyRestriction bool
}

type LocalityInfoSource interface {
	GetLocalityFlyInfo(Coordinate) (LocalityInfo, error)
}

type Client struct {
	WeatherInfoSource
	ZoneInfoSource
	//LocalityInfoSource
}
