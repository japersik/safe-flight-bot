package openstreetmap

import (
	"encoding/json"
	"fmt"
	"github.com/japersik/safe-flight-bot/internal/flyDataClient"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	getDataEndPoint = "https://nominatim.openstreetmap.org/reverse/"
)

type OpenStreetClient struct {
	webClient http.Client
}

func (c OpenStreetClient) GetLocalityFlyInfo(coordinate flyDataClient.Coordinate) (*flyDataClient.LocalityInfo, error) {
	url, err := url.Parse(getDataEndPoint)
	var req = &http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{"Content-Type": []string{"application/json"}},
	}

	q := req.URL.Query()
	q.Add("lat", strconv.FormatFloat(coordinate.Lat, 'f', 10, 64))
	q.Add("lon", strconv.FormatFloat(coordinate.Lng, 'f', 10, 64))
	q.Add("zoom", strconv.Itoa(14))
	q.Add("accept-language", "ru")
	q.Add("format", "jsonv2")
	req.URL.RawQuery = q.Encode()

	response, err := c.webClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	resp := struct {
		PlaceRank int `json:"place_rank"`
		Address   struct {
			Village      string `json:"village"`
			Town         string `json:"town"`
			City         string `json:"city"`
			Municipality string `json:"municipality"`
			County       string `json:"county"`
			State        string `json:"state"`
			ISO31662Lvl4 string `json:"ISO3166-2-lvl4"`
			Region       string `json:"region"`
			Postcode     string `json:"postcode"`
			Country      string `json:"country"`
			CountryCode  string `json:"country_code"`
		} `json:"address"`
	}{}
	if err = decoder.Decode(&resp); err != nil {
		return nil, err
	}

	ans := flyDataClient.LocalityInfo{}
	if resp.PlaceRank > 14 {
		ans.Name = fmt.Sprintf("%s %s %s %s", resp.Address.Village+resp.Address.Town+resp.Address.City, resp.Address.County, resp.Address.State, resp.Address.Country)
		ans.FlyRestriction = true
	} else {
		ans.Name = fmt.Sprintf("%s %s %s", resp.Address.County, resp.Address.State, resp.Address.Country)
		ans.FlyRestriction = false
	}
	return &ans, nil
}

func NewOpenStreetClient() *OpenStreetClient {
	return &OpenStreetClient{
		webClient: http.Client{
			Timeout: time.Second * 3,
		},
	}
}
