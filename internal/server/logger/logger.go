package logger

import "go.uber.org/zap"

// New создаёт логгер с указанным уровнем.
func New(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	return cfg.Build()
}
