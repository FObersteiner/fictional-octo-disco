package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log = zerolog.New(nil)
var logFileName = fmt.Sprintf("webserver_%v.log", time.Now().UTC().Format("20060102T150405Z"))

func init() {
	dst, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(dst, "log", logFileName),
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     3,
	}

	// log to console AND file.
	var writers []io.Writer
	writers = append(writers, zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05.000", // local time
	})
	writers = append(writers, lumberjackLogger)
	mw := io.MultiWriter(writers...)
	log = zerolog.New(mw).With().Caller().Timestamp().Logger()

	// log UTC, not local time
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000Z"
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
}

var indexTemplate = template.Must(template.ParseFiles("./tmpl/index.html"))

// method to fill template with most recent data
func handleData(w http.ResponseWriter, r *http.Request) {
	log.Info().Msgf("handler called from %v", r.RemoteAddr)

	measurements := getData()
	testdata := PageData{
		Title:   "Sensor Data",
		Data:    measurements,
		Updated: time.Now().Format(time.RFC3339),
	}

	err := indexTemplate.Execute(w, testdata)
	if err != nil {
		log.Error().Err(err)
	}
}

// method to obtain most recent data from database
func getData() []Measurement {
	var measurements = []Measurement{}

	// TODO: handle db client in separate goroutine
	client := influxdb2.NewClient(cfg.DBurl, cfg.DBtoken)
	defer client.Close()
	queryAPI := client.QueryAPI(cfg.DBorg)

	result, err := queryAPI.Query(context.Background(), fmt.Sprintf(
		`from(bucket: "%v")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "Arbeitszimmer" or r["_measurement"] == "Wohnzimmer")
  |> tail(n: 1)`, cfg.DBbucket),
	)

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
		if suffix, ok := units[m.Type]; ok {
			m.Type += ", " + suffix
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

var cfg = NewCfg()
var loc, _ = time.LoadLocation("Europe/Berlin")
var units = map[string]string{
	"T":  "°C",
	"rH": "%",
	"aH": "g/kg",
	"p":  "hPa",
}

func main() {
	// can supply path to config via cmd line arg
	var cfgPath string
	args := os.Args
	if len(args) > 1 {
		cfgPath = args[1]
	}

	err := cfg.Load(cfgPath)
	if err != nil {
		log.Error().Err(err)
		os.Exit(1)
	}

	fmt.Println(cfg)
	if strings.ToUpper(cfg.LogLevel) == "INFO" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	http.HandleFunc("/", handleData)

	fs := http.FileServer(http.Dir("./assets"))
	http.Handle("/assets/", http.StripPrefix("/assets", fs))

	log.Info().Msgf("listen & serve at localhost%v", cfg.ServePort)
	err = http.ListenAndServe(cfg.ServePort, nil)
	if err != nil {
		log.Error().Err(err)
	}
}

// for rpi:
// env GOOS=linux GOARCH=arm GOARM=5 go build

// scp -r /home/floo/Code/solltIchLueften/webserver/ floo@192.168.0.108:/home/floo/Documents/Go/
// scp -r /home/va6504/Code/Arduino/solltIchLueften/webserver/ floo@192.168.0.108:/home/floo/Documents/Go/
