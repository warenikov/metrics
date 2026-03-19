package models

import "fmt"

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

//func (m Metrics) ValueString() string {
//	switch m.MType {
//	case Gauge:
//		if m.Value != nil {
//			return utils.Float64ToString(*m.Value)
//		}
//	case Counter:
//		if m.Delta != nil {
//			return utils.Int64ToString(*m.Delta)
//		}
//	}
//	return ""
//}

func (m Metrics) ValueString() string {
	switch m.MType {
	case "gauge":
		if m.Value == nil {
			return "0"
		}
		return fmt.Sprintf("%v", *m.Value)
	case "counter":
		if m.Delta == nil {
			return "0"
		}
		return fmt.Sprintf("%d", *m.Delta)
	default:
		return ""
	}
}
