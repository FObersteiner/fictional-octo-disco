package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log         = zerolog.New(nil)
	logFileName = fmt.Sprintf("sensorlogger_%v.log", time.Now().UTC().Format("20060102T150405Z"))
)

// GetCwd tries to obtain the current working directory of the calling executable.
func GetCwd() (string, error) {
	// try to use the executable's path by default
	src, err := os.Executable()
	if err != nil {
		return "", err
	}

	// if os.Executable() appears to return a tmp build path, fall back to os.Getwd
	if strings.Contains(src, "go-build") {
		src, err = os.Getwd()
		if err != nil {
			return "", err
		}
		return src, nil
	}
	return filepath.Dir(src), nil
}

func init() {
	dst, err := GetCwd()
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

const (
	INTERVAL      = time.Duration(time.Minute)
	CHECKINTERVAL = time.Duration(time.Second)
	CSVSEP        = ";"
)

var (
	cfg     = NewCfg()
	sources = Sources{}
)

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

	if strings.ToUpper(cfg.LogLevel) == "INFO" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().Msg(cfg.DataSavePath)
	logpath := path.Clean(cfg.DataSavePath)
	if strings.HasPrefix(logpath, "~/") {
		usr, _ := user.Current()
		dir := usr.HomeDir
		logpath = filepath.Join(dir, logpath[2:])
	}
	if stat, err := os.Stat(logpath); err != nil || !stat.IsDir() {
		log.Error().Msgf("invalid path '%v'", logpath)
		os.Exit(64)
	}

	log.Info().Msgf("starting data collector, logging to '%v'", logpath)

	// declare sensor data sources
	for _, src := range cfg.Sources {
		addr, err := net.ResolveUDPAddr("udp", src.Address)
		if err != nil {
			log.Error().Err(err)
		}
		s := Source{
			Name: src.Name, ID: src.ID, Address: src.Address,
			UDPaddress: addr, Last_contact: time.Now().Add(-INTERVAL),
		}
		sources = append(sources, s)
	}

	// start data collector and handlers
	data := make(chan []byte)
	msgParserToCsv := make(chan message)
	msgCsvToDb := make(chan message)
	sigDone := make(chan struct{})

	// ctx, cancel := context.WithCancel(context.Background())
	ctx := context.Context(context.Background())

	go dataCollector(ctx, sources, data, sigDone)
	go dataParser(ctx, data, msgParserToCsv, sigDone)
	go handleCSVlog(ctx, logpath, msgParserToCsv, msgCsvToDb, sigDone)
	go handleDBupload(ctx, msgCsvToDb, sigDone)

	fmt.Println("press any key to exit...")
	fmt.Scanln()

	// stop goroutines via context and make sure they're closed before main stops
	// sjflasdfjsdal ftexie cancel()

	<-sigDone // data collector
	<-sigDone // msg parser
	<-sigDone // csv logger
	<-sigDone // db uploader

	log.Info().Msg("data collector graceful shutdown")
}

// for rpi:
// cp /home/floo/Code/Mixed/fictional-octo-disco/datalogger/src/cfg/config.toml /home/floo/Code/Mixed/fictional-octo-disco/datalogger/bin/cfg/config.toml
// env GOOS=linux GOARCH=arm GOARM=5 go build -o /home/floo/Code/Mixed/fictional-octo-disco/datalogger/bin/datalogger_rpi .

// to rpi:
// scp -r /home/floo/Code/Mixed/fictional-octo-disco/datalogger/bin/ floo@192.168.178.107:/home/floo/Documents/Go/datalogger
