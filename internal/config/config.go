package config

const (
	// PollInterval — интервал обновления метрик из runtime (секунды).
	PollInterval = 2
	// ReportInterval — интервал отправки метрик на сервер (секунды).
	ReportInterval = 10
	// DefaultServerURL — адрес сервера метрик по умолчанию.
	DefaultServerURL = "http://localhost:8080"
)
