package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

var cfg = NewCfg()
var loc, _ = time.LoadLocation("Europe/Berlin")
var units = map[string]string{
	"T":  "°C",
	"rH": "%",
	"aH": "g/kg",
	"p":  "hPa",
}

func main() {
	// ctx, cancel := context.WithCancel(context.Background())
	// capture control-C
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
		// cancel()
		log.Info().Msg("program terminated by os.Interrupt")
		os.Exit(0)
	}()

	// go makePlots(ctx)

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

	if strings.ToUpper(cfg.LogLevel) == "INFO" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	http.HandleFunc("/", serveData)
	log.Debug().Msg("created server for recent data")

	http.HandleFunc("/plots/test", plotserver)
	log.Debug().Msg("created server for test plot")

	// css directory
	fs := http.FileServer(http.Dir("./assets"))
	http.Handle("/assets/", http.StripPrefix("/assets", fs))

	// // plots directorxs/", http.StripPrefix("/plots", fs))

	log.Info().Msgf("listen & serve at localhost%v", cfg.ServePort)

	err = http.ListenAndServe(cfg.ServePort, nil)
	if err != nil {
		log.Error().Err(err)
	}
}

// for rpi:
// env GOOS=linux GOARCH=arm GOARM=5 go build

// rm -rf ./log && scp -r /home/floo/Code/Mixed/fictional-octo-disco/webserver/ floo@192.168.0.108:/home/floo/Documents/Go/
// rm -rf ./log && scp -r /home/va6504/Code/Arduino/solltIchLueften/webserver/ floo@192.168.0.108:/home/floo/Documents/Go/
