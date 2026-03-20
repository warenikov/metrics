package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr      string `env:"ADDRESS"`
	ReportInterval  int    `env:"REPORT_INTERVAL"`
	PollInterval    int    `env:"POLL_INTERVAL"`
	LogLevel        string `env:"LOG_LEVEL"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func LoadConfig() *Config {
	//устанавливаем значение по умолчанию
	cfg := &Config{
		ServerAddr:      "localhost:8080",
		ReportInterval:  10,
		PollInterval:    2,
		LogLevel:        "info",
		StoreInterval:   300,
		FileStoragePath: "storage.txt",
		Restore:         true,
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
	flag.IntVar(&cfg.ReportInterval, "rep", cfg.ReportInterval, "Reporting interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Polling interval in seconds")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File to storage")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore metrics from file")
	flag.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "Log level")
	flag.Parse()
}

func loadFromEnv(cfg *Config) {
	err := env.Parse(cfg)
	if err != nil {
		log.Printf("Can't parse config from os: %s ", err)
	}
}
