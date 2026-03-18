package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr     string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	LogLevel       string `env:"LOG_LEVEL"`
}

func LoadConfig() *Config {
	//устанавливаем значение по умолчанию
	cfg := &Config{
		ServerAddr:     ":8080",
		ReportInterval: 10,
		PollInterval:   2,
		LogLevel:       "info",
	}

	//нужно получить параметры для запуска приложения в таком приоритере:
	//1. Если указана переменная окружения, то используется она.
	loadFromEnv(cfg)
	//2. Если нет переменной окружения, но есть аргумент командной строки (флаг), то используется он.
	loadFromCLI(cfg)
	//3. Если нет ни переменной окружения, ни флага, то используется значение по умолчанию, которое установлено изначально.

	return cfg
}

func loadFromCLI(cfg *Config) {
	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP address")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Reporting interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Polling interval in seconds")
	flag.StringVar(&cfg.ServerAddr, "l", cfg.LogLevel, "Log level")

	flag.Parse()
}

func loadFromEnv(cfg *Config) {
	err := env.Parse(cfg)
	if err != nil {
		log.Printf("Can't parse config from os: %s ", err)
	}
}
