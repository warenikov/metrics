package models

// AppError — общая структура для ошибок
type AppError struct {
	Message    string
	StatusCode int
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrInvalidMetricType   = &AppError{Message: "invalid metric type", StatusCode: 400}
	ErrInvalidValue        = &AppError{Message: "invalid metric value", StatusCode: 400}
	ErrMetricNotFound      = &AppError{Message: "metric not found", StatusCode: 404}
	ErrInternalServerError = &AppError{Message: "internal server error", StatusCode: 500}
)
