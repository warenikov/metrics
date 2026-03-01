package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Config struct {
	ServerAddr     string `json:"server_addr"`
	ReportInterval int    `json:"report_interval"`
	PollInterval   int    `json:"poll_interval"`
}

func LoadConfig() *Config {
	//устанавливаем значение по умолчанию
	cfg := &Config{
		ServerAddr:     "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
	}

	//получаем конфиг из файла json
	err := loadFromFile(cfg, "config.json")
	if err != nil {
		//TODO: прикрутить логировние ошибки тут
		fmt.Println("Ошибка парсинга конфига:", err)
	}

	//получаем конфиг из командной строки
	loadFromCLI(cfg)

	return cfg
}

func loadFromFile(cfg *Config, path string) error {
	if file, err := os.Open(path); err == nil {
		defer file.Close()
		if err = json.NewDecoder(file).Decode(cfg); err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}

func loadFromCLI(cfg *Config) {
	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP address")
	flag.IntVar(&cfg.ReportInterval, "ri", cfg.ReportInterval, "Reporting interval in seconds")
	flag.IntVar(&cfg.PollInterval, "pi", cfg.PollInterval, "Polling interval in seconds")

	flag.Parse()
}
