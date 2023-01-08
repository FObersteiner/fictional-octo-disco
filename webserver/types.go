package main

import (
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel  string `yaml:"Log_Level"`
	ServePort string `yaml:"Serve_Port"`
	DBtoken   string `yaml:"DB_Token"`
	DBorg     string `yaml:"DB_Org"`
	DBurl     string `yaml:"DB_Url"`
	DBbucket  string `yaml:"DB_Bucket"`
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

// struct to feed most recent sensor data to template
type Measurement struct {
	Name, Time, Type, Value string
}

type PageData struct {
	Data    []Measurement
	Updated string
}
