package avtmClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	currentWeatherEndPoint  = "https://map.avtm.center/app/public/services/weather-current"
	forecastWeatherEndPoint = "https://map.avtm.center/app/public/services/weather-forecast"
	checkConditionsEndPoint = "https://map.avtm.center/app/flight-check/check-conditions"
)

type AvmtClient struct {
	webClient http.Client
}

func NewAvmtClient() *AvmtClient {
	return &AvmtClient{
		http.Client{
			Timeout: time.Second * 3,
		},
	}
}

//GetCurrentWeather receives current  weather as flyDataClient.CurrentWeatherData form Avmt api.
func (c AvmtClient) GetCurrentWeather(coordinate flyDataClient.Coordinate) (*flyDataClient.CurrentWeatherData, error) {
	url, err := url.Parse(currentWeatherEndPoint)
	var req = &http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}
	q := req.URL.Query()
	q.Add("lat", strconv.FormatFloat(coordinate.Lat, 'f', 10, 64))
	q.Add("lng", strconv.FormatFloat(coordinate.Lng, 'f', 10, 64))
	q.Add("lang", "ru")

	req.URL.RawQuery = q.Encode()
	response, err := c.webClient.Do(req)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(response.Body)
	ans := &flyDataClient.CurrentWeatherData{}
	if err = decoder.Decode(ans); err != nil {
		return nil, err
	}
	return ans, nil
}

//type JSONTime struct {
//	time.Time
//}

const ctLayout = "2006-01-02T15:04:05.000"

//func (ct *JSONTime) UnmarshalJSON(b []byte) (err error) {
//	s := strings.Trim(string(b), "\"")
//	if s == "null" {
//		ct.Time = time.Time{}
//		return
//	}
//	ct.Time, err = time.Parse(ctLayout, s)
//	return
//}

type JSONWeatherData struct {
	PrecipProb  float64
	Temperature float64
	WindSpeed   float64
	WindDeg     float64
	Pressure    int
	Humidity    int
	Visibility  int
	Clouds      int
	Timestamp   string
}

//castToWeatherData
func (jwd JSONWeatherData) castToWeatherData(loc *time.Location) flyDataClient.WeatherData {
	t, _ := time.ParseInLocation(ctLayout, jwd.Timestamp, loc)
	return flyDataClient.WeatherData{
		PrecipProb:  jwd.PrecipProb,
		Temperature: jwd.Temperature,
		WindSpeed:   jwd.WindSpeed,
		WindDeg:     jwd.WindDeg,
		Pressure:    jwd.Pressure,
		Humidity:    jwd.Humidity,
		Visibility:  jwd.Visibility,
		Clouds:      jwd.Clouds,
		Timestamp:   t,
	}
}

//GetForecastWeather receives forecast weather as flyDataClient.WeatherData form Avmt api.
func (c AvmtClient) GetForecastWeather(coordinate flyDataClient.Coordinate) (*flyDataClient.WeatherData, error) {
	url, err := url.Parse(forecastWeatherEndPoint)
	var req = &http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}

	q := req.URL.Query()
	q.Add("lat", strconv.FormatFloat(coordinate.Lat, 'f', 10, 64))
	q.Add("lng", strconv.FormatFloat(coordinate.Lng, 'f', 10, 64))
	req.URL.RawQuery = q.Encode()

	response, err := c.webClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)

	resp := struct {
		TimeZoneName    string
		WeatherForecast struct {
			Current JSONWeatherData   `json:"current"`
			Hourly  []JSONWeatherData `json:"hourly"`
		} `json:"weatherForecast"`
	}{}
	if err = decoder.Decode(&resp); err != nil {
		return nil, err
	}
	ans := struct {
		WeatherForecast struct {
			Current flyDataClient.WeatherData   `json:"current"`
			Hourly  []flyDataClient.WeatherData `json:"hourly"`
		} `json:"weatherForecast"`
	}{
		WeatherForecast: struct {
			Current flyDataClient.WeatherData   `json:"current"`
			Hourly  []flyDataClient.WeatherData `json:"hourly"`
		}{},
	}
	tl, _ := time.LoadLocation(resp.TimeZoneName)
	ans.WeatherForecast.Current = resp.WeatherForecast.Current.castToWeatherData(tl)
	for _, data := range resp.WeatherForecast.Hourly {
		fmt.Println(data.Timestamp)
		ans.WeatherForecast.Hourly = append(ans.WeatherForecast.Hourly, data.castToWeatherData(tl))

	}
	return nil, nil
}

type JSONCheckConditions struct {
	// ??
	CheckTime int `json:"checkTime"`
	// Day or Night
	DaylightHours    bool `json:"daylightHours"`
	HasIntersections bool `json:"hasIntersections"`
	//In Russia (Foreign territories not supported )
	IntoCountryBoundary bool `json:"intoCountryBoundary"`
	//(20 km Boundary Zone)
	NearBoundaryZone    bool                `json:"nearBoundaryZone"`
	Permanent           bool                `json:"permanent"`
	PolarDayOrNight     bool                `json:"polarDayOrNight"`
	LocalTimeInLocation string              `json:"localTimeInLocation"`
	Sunrise             string              `json:"sunrise"`
	Sunset              string              `json:"sunset"`
	Zones               map[string]ZoneInfo `json:"map"`
}

type ZoneInfo struct {
	Inactive           []interface{} `json:"inactive"`
	Active             []interface{} `json:"active"`
	IntersectionCodes  []string      `json:"intersectionCodes"`
	CompletedWithError bool          `json:"completedWithError"`
	FullTime           int           `json:"fullTime"`
	ComputeTime        int           `json:"computeTime"`
	SelectTime         int           `json:"selectTime"`
}

//type IntersectionZoneInfo{
//
//}

//CheckConditions  receives fly zone Conditions form Avmt api.
func (c AvmtClient) CheckConditions(coordinate flyDataClient.Coordinate, radius int) (int, error) {
	type Geometry struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}

	type Area struct {
		Geometry   `json:"geometry"`
		Properties struct{} `json:"properties"`
		Type       string   `json:"type"`
	}
	type ccReq struct {
		Area          `json:"area"`
		Altitude      int `json:"altitude"`
		DurationHours int `json:"durationHours"`
	}

	n := 8
	coordinates := make([][2]float64, 0, n)
	for i := 0; i < n; i++ {
		newCoord := circleCoordinate(coordinate, radius, 360/n*i)
		fmt.Println(newCoord)
		coordinates = append(coordinates, [2]float64{newCoord.Lng, newCoord.Lat})
	}
	geometry := Geometry{
		Type:        "Polygon",
		Coordinates: [][][2]float64{coordinates},
	}
	area := Area{
		Geometry:   geometry,
		Properties: struct{}{},
		Type:       "Feature",
	}
	reqArg := ccReq{
		Area:          area,
		Altitude:      150,
		DurationHours: 1,
	}
	data, _ := json.Marshal(reqArg)
	r := bytes.NewReader(data)
	resp, err := c.webClient.Post(checkConditionsEndPoint, "application/json", r)
	if err != nil {
		return 0, err
	}
	decoder := json.NewDecoder(resp.Body)
	ans := &JSONCheckConditions{}
	decoder.Decode(&ans)
	for s, i := range ans.Zones {
		fmt.Println(s)
		fmt.Println(i.IntersectionCodes)
		fmt.Println(i.Inactive)
	}
	fmt.Println(ans)
	return 0, nil
}
func circleCoordinate(coordinate flyDataClient.Coordinate, radius int, andreDeg int) flyDataClient.Coordinate {
	angle := float64(andreDeg) * math.Pi * 2 / 360
	dx := float64(radius) * math.Cos(angle)
	dy := float64(radius) * math.Sin(angle)
	ans := flyDataClient.Coordinate{
		Lat: coordinate.Lat + (180/math.Pi)*(dy/6378137),
		Lng: coordinate.Lng + (180/math.Pi)*(dx/6378137)/math.Cos(coordinate.Lat*math.Pi/180),
	}
	return ans
}
