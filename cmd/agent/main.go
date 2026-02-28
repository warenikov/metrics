package main

import (
	"metrics/internal/agent"
	"metrics/internal/config"
)

func main() {
	cfg := config.LoadConfig()
	agent := agent.NewMetricaAgent(cfg)
	go agent.Run()

	select {}
}
