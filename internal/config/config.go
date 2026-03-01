package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Config struct {
	ServerAddr     string `json:"server_addr"`
	ServerPort     string `json:"server_port"`
	ReportInterval int    `json:"report_interval"`
	PollInterval   int    `json:"poll_interval"`
}

func LoadConfig() *Config {
	//устанавливаем значение по умолчанию
	cfg := &Config{
		ServerAddr:     "localhost",
		ServerPort:     "8080",
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
	file, err := os.Open(path)

	if err != nil {
		return err
	}
	defer file.Close()

	if file, err = os.Open(path); err == nil {
		defer file.Close()
		if err = json.NewDecoder(file).Decode(cfg); err != nil {
			return err
		}
		return nil
	}
	return err
}

func loadFromCLI(cfg *Config) {
	flag.StringVar(&cfg.ServerAddr, "a", "localhost", "HTTP address")
	flag.StringVar(&cfg.ServerPort, "p", "8080", "HTTP port")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Reporting interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Polling interval in seconds")

	flag.Parse()

}
