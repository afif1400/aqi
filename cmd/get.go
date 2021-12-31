// Package cmd /*
package cmd

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the Air Quality Index",
	Long: `Get will fetch the Air Quality Index of specify geographical location
which can be specified through multiple flags and arguments.`,
	Run: getAqi,
	// Args: cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings for your command
	getCmd.PersistentFlags().StringP("city", "c", "", "--city=<city name> or -c <city name>")
	getCmd.PersistentFlags().StringP("postal", "p", "", "--postal=<postal code> or -p <postal code>")
	getCmd.PersistentFlags().StringP("country", "o", "", "--country=<country name> or -o <country name>")
	getCmd.PersistentFlags().StringP("latitude", "l", "", "--latitude=<latitude> or -l <latitude>")
	getCmd.PersistentFlags().StringP("longitude", "g", "", "--longitude=<longitude> or -g <longitude>")
}

type ApiResponse struct {
	Message  string    `json:"message"`
	Stations []Station `json:"stations"`
}

type Station struct {
	CO          float64 `json:"CO"`
	NO2         float64 `json:"NO2"`
	OZONE       float64 `json:"OZONE"`
	PM10        float64 `json:"PM10"`
	PM25        float64 `json:"PM25"`
	CountryCode string  `json:"countryCode"`
	Division    string  `json:"division"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lng"`
	PostalCode  string  `json:"postalCode"`
	City        string  `json:"city"`
	Place       string  `json:"placeName"`
	State       string  `json:"state"`
	UpdatedAt   string  `json:"updatedAt"`
	AQI         float64 `json:"AQI"`
	AqiInfo     AqiInfo
}

type Aqi struct {
	City      string  `json:"city"`
	Place     string  `json:"placeName"`
	State     string  `json:"state"`
	UpdatedAt string  `json:"updatedAt"`
	AQI       float64 `json:"AQI"`
	AqiInfo   AqiInfo
}

type AqiInfo struct {
	Pollutant     string  `json:"pollutant"`
	Concentration float64 `json:"concentration"`
	Category      string  `json:"category"`
}

func getAqi(cmd *cobra.Command, args []string) {

	var (
		city, postal, country, latitude, longitude string
	)

	city = cmd.Flag("city").Value.String()
	postal = cmd.Flag("postal").Value.String()
	country = cmd.Flag("country").Value.String()
	latitude = cmd.Flag("latitude").Value.String()
	longitude = cmd.Flag("longitude").Value.String()

	var url string
	if city != "" && postal == "" && country == "" && latitude == "" && longitude == "" {
		url = "http://localhost:3000/api?city=" + city
	} else if postal != "" && country != "" && latitude == "" && longitude == "" {
		url = "http://localhost:3000/api?postal=" + postal + "&country=" + country
	} else if latitude != "" && longitude != "" {
		url = "http://localhost:3000/api?lat=" + latitude + "&lng=" + longitude
	} else {
		log.Fatal("please specify the location by using any of the following flags: --city, --postal, --country, --latitude, --longitude")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("error occured: %v", err)
		}
	}(res.Body)

	body, _ := ioutil.ReadAll(res.Body)

	aqi := ApiResponse{}
	err = json.Unmarshal(body, &aqi)
	if err != nil {
		log.Fatalf("unable to unmarshell the response : %v", err)
	}

	if len(aqi.Stations) == 0 {
		log.Fatal("no stations found")
	}

	for _, station := range aqi.Stations {
		aqi := Aqi{
			City:      station.City,
			Place:     station.Place,
			State:     station.State,
			UpdatedAt: station.UpdatedAt,
			AQI:       station.AQI,
			AqiInfo:   station.AqiInfo,
		}
		json, err := json.Marshal(aqi)
		if err != nil {
			log.Fatalf("unable to marshall the response : %v", err)
		}
		log.Println(string(json))
		dataUi(aqi.AQI, station)
	}
}

func dataUi(aqi float64, station Station) {
	aqiInfo := station.AqiInfo
	// use termui to display the data

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}

	defer ui.Close()

	// create a guage for the AQI
	g := widgets.NewGauge()
	g.Percent = int((aqi / 500) * 100)
	g.SetRect(0, 0, 50, 5)
	g.Title = "Air Quality Index = " + strconv.Itoa(int(aqi))
	// set the color based on the category
	switch aqiInfo.Category {
	case "Good":
		g.BarColor = ui.ColorGreen
	case "Moderate":
		g.BarColor = ui.ColorYellow
	case "Unhealthy for Sensitive Groups":
		g.BarColor = ui.ColorRed
	case "Unhealthy":
		g.BarColor = ui.ColorRed
	case "Very Unhealthy":
		g.BarColor = ui.ColorRed
	case "Hazardous":
		g.BarColor = ui.ColorRed
	}

	//slim list of all other items
	list := widgets.NewList()
	list.Rows = []string{
		"Pollutant: " + aqiInfo.Pollutant,
		"Concentration: " + strconv.Itoa(int(aqiInfo.Concentration)),
		"Category: " + aqiInfo.Category,
	}
	// set list to right of gauge
	list.SetRect(50, 0, 100, 5)

	// also display city, state, placeName and updatedAt in a list
	list2 := widgets.NewList()
	list2.Rows = []string{
		"City: " + station.City,
		"State: " + station.State,
		"Place: " + station.Place,
		"Updated At: " + station.UpdatedAt,
	}
	// set list above gauge
	list2.SetRect(0, 5, 50, 10)
	ui.Render(g, list, list2)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}
