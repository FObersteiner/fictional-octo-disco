package main

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

const day = 24 * time.Hour

func check(e error) {
	if e != nil {
		log.Error().Err(e)
	}
}

// FloatIsClose checks if two floating point numbers are equal within tolerance.
func FloatIsClose(have, want, tolerance float64) bool {
	diff := have - want
	if diff < 0 {
		diff *= -1
	}
	return diff <= tolerance
}

// DateDifferent checks if the date of time.Time "now" is greater then that of "prev"
func DateDifferent(now, prev *time.Time) bool {
	return !(now.Truncate(day).Equal(prev.Truncate(day)))
}

// PrependDate prepends the current date to a string and separates it with an underscore,
// YYYYMMDD_, from a given time.Time t
func PrependDate(t *time.Time, s string) string {
	if t.Location() == time.UTC {
		return t.Format("20060102Z07:00_") + s
	}
	return t.Format("20060102_") + s
}

// dataParser receives messsages from the dataCollector.
// Parses the JSON to message type and adds timestamp
func dataParser(ctx context.Context, data <-chan []byte, msgOut chan<- message, sigDone chan<- struct{}) {
	for {
		select {
		case recv := <-data:
			msg := new(message)
			err := json.Unmarshal(recv, &msg)
			check(err)
			msg.Timestamp = time.Now().UTC()
			if name, ok := id2Name[msg.ID]; ok {
				msg.Name = name
			} else {
				msg.Name = "UNKNOWN"
			}
			// verify data is valid; assum invalid if both temperature and rel.Hum == 0
			if FloatIsClose(float64(msg.Temperature), 0.0, 1e-5) && FloatIsClose(float64(msg.RelHum), 0.0, 1e-5) {
				log.Error().Msgf("could not parse string '%v'", string(recv))
			} else {
				log.Info().Msg(msg.StringShort())
				msg.AbsHum = float32(calcAbsHum(float64(msg.RelHum), float64(msg.Temperature)))
				msgOut <- *msg
			}

		case <-ctx.Done():
			log.Debug().Msg("parser closing")
			sigDone <- struct{}{}
			return
		}
	}
}

// makeLogfile creates a new csv file to log data for one day
func makeLogfile(logpath string, now *time.Time) (*os.File, error) {
	logfile := path.Join(logpath, PrependDate(now, "sensordata.csv"))
	f, fileErr := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	check(fileErr)
	if fileErr == nil {
		fi, err := os.Stat(logfile)
		check(err)
		if err == nil {
			if fi.Size() < 5 { // only write the header if the file is empty
				_, err := f.WriteString(NewMsg().CsvHeader(CSVSEP))
				check(err)
			}
		}
	}
	return f, fileErr
}

// handleCSVlog logs parsed messages to a csv table.
// Forwards the data to database logger
func handleCSVlog(ctx context.Context, logpath string,
	msgIn <-chan message, msgOut chan<- message, sigDone chan<- struct{}) {
	now := time.Now().UTC()
	f, fileErr := makeLogfile(logpath, &now)
	for {
		select {
		case recv := <-msgIn:
			log.Debug().Msgf("logging '%v'", strings.Trim(recv.StringCsv(CSVSEP), "\n"))
			// to csv
			if newNow := time.Now().UTC(); DateDifferent(&newNow, &now) {
				// date has changed, make a new logfile
				if fileErr == nil {
					f.Close() // close existing logfile only if it was created sucessfully before
				}
				now = newNow
				f, fileErr = makeLogfile(logpath, &now)
			}
			if fileErr == nil {
				n, err := f.WriteString(recv.StringCsv(CSVSEP))
				check(err)
				log.Debug().Msgf("wrote %v bytes to logfile", n)
			}
			// forward to handleDBupload
			msgOut <- recv
		case <-ctx.Done():
			log.Debug().Msg("csv handler closing")
			f.Close()
			sigDone <- struct{}{}
			return
		}
	}
}

// handleDBupload sends data to the database
func handleDBupload(ctx context.Context, msgIn <-chan message, sigDone chan<- struct{}) {
	// Create a new client using an InfluxDB server base URL and an authentication token
	// and set batch size to 20
	client := influxdb2.NewClientWithOptions(cfg.DBurl, cfg.DBtoken,
		influxdb2.DefaultOptions().SetBatchSize(20))
	// Get non-blocking write client
	writeAPI := client.WriteAPI(cfg.DBorg, cfg.DBbucket)
	for {
		select {
		case recv := <-msgIn:
			log.Debug().Msgf("DB logger received msg %v", recv)
			writeAPI.WritePoint(recv.ToInfluxPoint())
		case <-ctx.Done():
			log.Debug().Msg("db uploader closing")
			writeAPI.Flush()
			client.Close()
			sigDone <- struct{}{}
			return
		}
	}
}
