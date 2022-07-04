package avtmClient

import (
	"bytes"
	"encoding/json"
	"github.com/japersik/safe-flight-bot/logger"
	"github.com/japersik/safe-flight-bot/model"
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

type AvtmClient struct {
	webClient http.Client
}

func NewAvtmClient() *AvtmClient {
	return &AvtmClient{
		http.Client{
			Timeout: time.Second * 3,
		},
	}
}

//GetCurrentWeather receives current  weather as flyDataClient.CurrentWeatherData form Avmt api.
func (c AvtmClient) GetCurrentWeather(coordinate model.Coordinate) (*model.CurrentWeatherData, error) {
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
	ans := &model.CurrentWeatherData{}
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
func (jwd JSONWeatherData) castToWeatherData(loc *time.Location) model.WeatherData {
	t, _ := time.ParseInLocation(ctLayout, jwd.Timestamp, loc)
	return model.WeatherData{
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
func (c AvtmClient) GetForecastWeather(coordinate model.Coordinate) (*model.WeatherForecast, error) {
	logger.InfoF("getting weather forecast at point (%f, %f)\n", coordinate.Lat, coordinate.Lng)
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
	ans := model.WeatherForecast{
		Current: model.WeatherData{},
		Hourly:  make([]model.WeatherData, 0, len(resp.WeatherForecast.Hourly)),
	}

	tl, _ := time.LoadLocation(resp.TimeZoneName)
	ans.Current = resp.WeatherForecast.Current.castToWeatherData(tl)
	for _, data := range resp.WeatherForecast.Hourly {
		ans.Hourly = append(ans.Hourly, data.castToWeatherData(tl))

	}
	return &ans, nil
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
	Inactive           []map[string]interface{} `json:"inactive"`
	Active             []map[string]interface{} `json:"active"`
	IntersectionCodes  []string                 `json:"intersectionCodes"`
	CompletedWithError bool                     `json:"completedWithError"`
	FullTime           int                      `json:"fullTime"`
	ComputeTime        int                      `json:"computeTime"`
	SelectTime         int                      `json:"selectTime"`
}

func (loc *JSONCheckConditions) castJSONtoCondition() model.Condition {
	ans := model.Condition{
		DaylightHours:       loc.DaylightHours,
		HasIntersections:    loc.HasIntersections,
		IntoCountryBoundary: loc.IntoCountryBoundary,
		NearBoundaryZone:    loc.NearBoundaryZone,
		Permanent:           loc.Permanent,
		PolarDayOrNight:     loc.PolarDayOrNight,
		LocalTimeInLocation: "",
		Sunrise:             loc.Sunrise,
		Sunset:              loc.Sunset,
		ActiveZones:         []string{},
		InactiveZones:       []string{},
	}
	for _, info := range loc.Zones {
		ans.ActiveZones = append(ans.ActiveZones, info.IntersectionCodes...)
		for _, m := range info.Inactive {
			if i, ok := m["code"]; ok {

				if st, ok := i.(string); ok {
					ans.InactiveZones = append(ans.InactiveZones, st)
				}
			}
		}

	}
	return ans
}

//CheckConditions  receives fly zone Conditions form Avmt api.
func (c AvtmClient) CheckConditions(coordinate model.Coordinate, radius int) (model.Condition, error) {
	logger.Info("getting information about fly zones at point (%f, %f)\n", coordinate.Lat, coordinate.Lng)
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
		return model.Condition{}, err
	}
	decoder := json.NewDecoder(resp.Body)
	ans := &JSONCheckConditions{}
	decoder.Decode(&ans)
	//for s, i := range ans.Zones {
	//	fmt.Println(s)
	//	fmt.Println(i.Active)
	//	fmt.Println(i.Inactive)
	//}
	//fmt.Println(ans)
	return ans.castJSONtoCondition(), nil
}
func circleCoordinate(coordinate model.Coordinate, radius int, andreDeg int) model.Coordinate {
	angle := float64(andreDeg) * math.Pi * 2 / 360
	dx := float64(radius) * math.Cos(angle)
	dy := float64(radius) * math.Sin(angle)
	ans := model.Coordinate{
		Lat: coordinate.Lat + (180/math.Pi)*(dy/6378137),
		Lng: coordinate.Lng + (180/math.Pi)*(dx/6378137)/math.Cos(coordinate.Lat*math.Pi/180),
	}
	return ans
}
