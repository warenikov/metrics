package audit

import (
	"context"

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

// Broker рассылает событие всем зарегистрированным наблюдателям.
type Broker struct {
	observers []Observer
}

// NewBroker создаёт брокер с переданными наблюдателями.
func NewBroker(observers ...Observer) *Broker {
	return &Broker{observers: observers}
}

// Emit отправляет событие всем наблюдателям; ошибки логирует, не возвращает.
func (b *Broker) Emit(ctx context.Context, event AuditEvent) {
	for _, o := range b.observers {
		if err := o.Notify(ctx, event); err != nil {
			logger.Log.Error("audit observer error", zap.Error(err))
		}
	}
}
