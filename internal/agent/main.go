package agent

import (
	"metrics/internal/config"
	"runtime"
)

func NewAgent(cfg *config.Config) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
}
