package audit

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"metrics/internal/logger"
)

// AuditEvent описывает одно событие аудита.
type AuditEvent struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer — получатель событий аудита.
type Observer interface {
	Notify(ctx context.Context, event AuditEvent) error
}

const (
	brokerWorkers   = 8
	brokerQueueSize = 256
)

// Broker рассылает события наблюдателям через пул воркеров.
// Emit не блокирует вызывающего: событие кладётся в буферизованный канал.
// При заполненном буфере событие дропается с предупреждением в лог.
type Broker struct {
	observers []Observer
	queue     chan AuditEvent
	wg        sync.WaitGroup
}

// NewBroker создаёт брокер с переданными наблюдателями и запускает воркеры.
func NewBroker(observers ...Observer) *Broker {
	b := &Broker{
		observers: observers,
		queue:     make(chan AuditEvent, brokerQueueSize),
	}
	for range brokerWorkers {
		b.wg.Add(1)
		go b.worker()
	}
	return b
}

func (b *Broker) worker() {
	defer b.wg.Done()
	for event := range b.queue {
		for _, o := range b.observers {
			if err := o.Notify(context.Background(), event); err != nil {
				logger.Log.Error("audit observer error", zap.Error(err))
			}
		}
	}
}

// Emit ставит событие в очередь и немедленно возвращает управление.
// Если очередь заполнена — событие дропается с предупреждением.
func (b *Broker) Emit(_ context.Context, event AuditEvent) {
	select {
	case b.queue <- event:
	default:
		logger.Log.Warn("audit broker queue full, dropping event")
	}
}

// Close завершает все воркеры, дожидаясь обработки оставшихся событий.
// Вызывать при завершении приложения и в конце тестов перед assert-ами.
func (b *Broker) Close() {
	close(b.queue)
	b.wg.Wait()
}
