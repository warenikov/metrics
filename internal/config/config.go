package config

import (
	"flag"
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

	//получаем конфиг из командной строки
	loadFromCLI(cfg)

	return cfg
}

func loadFromCLI(cfg *Config) {
	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP address")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Reporting interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Polling interval in seconds")

	flag.Parse()
}
