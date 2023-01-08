package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

var indexTemplate = template.Must(template.ParseFiles("./tmpl/index.html"))

func queryData(fluxquery string) (*api.QueryTableResult, error) {
	client := influxdb2.NewClient(cfg.DBurl, cfg.DBtoken)
	defer client.Close()
	queryAPI := client.QueryAPI(cfg.DBorg)
	return queryAPI.Query(context.Background(), fluxquery)
}

// method to fill template with most recent data
func serveData(w http.ResponseWriter, r *http.Request) {
	log.Info().Msgf("handler called from %v", r.RemoteAddr)

	measurements := getRecentData()
	data := PageData{
		Data:    measurements,
		Updated: time.Now().Format(time.RFC3339),
	}

	err := indexTemplate.Execute(w, data)
	if err != nil {
		log.Error().Err(err)
	}
}

// method to obtain most recent data from database
func getRecentData() []Measurement {
	var measurements = []Measurement{}

	measurementParts := []string{}
	for _, m := range cfg.Measurements {
		measurementParts = append(measurementParts,
			fmt.Sprintf("r[\"_measurement\"] == \"%v\"", m))
	}
	measurementFilter := strings.Join(measurementParts, " or ")
	result, err := queryData(fmt.Sprintf(
		`from(bucket: "%v")
  |> range(start: -1h)
  |> filter(fn: (r) => %v)
  |> tail(n: 1)`,
		cfg.DBbucket, measurementFilter))

	if err != nil {
		log.Error().Err(err)
		return measurements
	}

	for result.Next() {
		m := Measurement{
			Name:  result.Record().Measurement(),
			Time:  result.Record().Time().In(loc).Format("2006-01-02 15:04:05 MST"),
			Type:  result.Record().Field(),
			Value: fmt.Sprintf("%.2f", result.Record().Value()),
		}
		if unit, ok := units[m.Type]; ok {
			m.Type += ", " + unit + ":"
		}
		measurements = append(measurements, m)
	}
	if result.Err() != nil {
		log.Error().Msgf("Query error: %s\n", result.Err().Error())
	}

	sort.SliceStable(measurements, func(i, j int) bool {
		return measurements[i].Name < measurements[j].Name
	})
	return measurements
}
