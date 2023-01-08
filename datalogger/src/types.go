package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel     string `yaml:"Log_Level"`     // zerolog logger level
	DataSavePath string `yaml:"Path_Data_Log"` // where to save csvs
	LogToDB      bool   `yaml:"Log_to_DB"`
	DBtoken      string `yaml:"DB_Token"`
	DBorg        string `yaml:"DB_Org"`
	DBurl        string `yaml:"DB_Url"`
	DBbucket     string `yaml:"DB_Bucket"`
}

func (c *Config) Load(cfgPath string) error {
	if cfgPath == "" {
		cfgPath += "./cfg/config.yml"
	}

	cfgPath = path.Clean(cfgPath)
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return err
	}
	return nil
}

func NewCfg() *Config {
	return &Config{}
}

// arduino represents one micro-controller that should be queried
// for sensor data.
type arduino struct {
	id           uint8
	name         string
	address      string
	UDPaddress   *net.UDPAddr
	last_contact time.Time
}

// message represents the data in a UDP reply from an arduino.
type message struct {
	Timestamp   time.Time
	ID          uint8 `json:"ID"`
	Name        string
	Temperature float32 `json:"T"`
	RelHum      float32 `json:"rH"`
	AbsHum      float32 `json:"aH"`
	Pressure    float32 `json:"p"`
}

func NewMsg() *message {
	return &message{}
}

// String builds the full string repr of a message
func (m *message) String() string {
	repr := fmt.Sprintf("Timestamp: %v, ", m.Timestamp.Format("2006-01-02 15:04:05 -07:00"))
	repr += fmt.Sprintf("Name: %v, ", m.Name)
	repr += fmt.Sprintf("ID: %v\n", m.ID)
	repr += fmt.Sprintf("Temperature: %.2f °C, ", m.Temperature)
	repr += fmt.Sprintf("rel.Humidity: %.2f %%, ", m.RelHum)
	repr += fmt.Sprintf("abs.Humidity: %.2f g/kg", m.AbsHum)
	if m.Pressure > 500. {
		repr += fmt.Sprintf(", Pressure: %.2f hPa", m.Pressure)
	}
	return repr
}

// StringShort builds a shorter repr of a message
func (m *message) StringShort() string {
	repr := fmt.Sprintf("%v: ", m.Timestamp.Format("2006-01-02 15:04:05"))
	repr += fmt.Sprintf("%-14v ", m.Name)
	repr += fmt.Sprintf("%.2f°C, ", m.Temperature)
	repr += fmt.Sprintf("%.2f%% rH, ", m.RelHum)
	repr += fmt.Sprintf("%.2f g/kg aH", m.AbsHum)
	if m.Pressure > 500. {
		repr += fmt.Sprintf(", %.2f hPa", m.Pressure)
	}
	return repr
}

// StringCsv builds a csv repr of a message
func (m *message) StringCsv(sep string) string {
	repr := m.Timestamp.Format(time.RFC3339) + sep
	repr += fmt.Sprintf("%v%v", m.ID, sep)
	repr += m.Name + sep
	repr += fmt.Sprintf("%.3f%v", m.Temperature, sep)
	repr += fmt.Sprintf("%.3f%v", m.RelHum, sep)
	repr += fmt.Sprintf("%.3f%v", m.AbsHum, sep)
	repr += fmt.Sprintf("%.3f", m.Pressure)
	return repr + "\n"
}

// CsvHeader builds a csv header line for logging messages
func (m *message) CsvHeader(sep string) string {
	return strings.Join([]string{"datetime", "id", "name", "temp_degC", "relHum_%", "absHum_gkg", "pres_hPa"}, sep) + "\n"
}

// ToInfluxPoint converts message to influxDB point
func (m *message) ToInfluxPoint() *write.Point {
	p := influxdb2.NewPointWithMeasurement(m.Name).
		AddTag("id", fmt.Sprintf("%d", m.ID)).
		AddField("T", m.Temperature).
		AddField("rH", m.RelHum).
		AddField("aH", m.AbsHum).
		SetTime(m.Timestamp)
	if m.Pressure > 250. {
		p.AddField("p", m.Pressure)
	}
	return p
}
