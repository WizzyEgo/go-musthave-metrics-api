package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lw := &loggingResponseWriter{
			ResponseWriter: w,
			data:           &responseData{},
		}

		next.ServeHTTP(lw, r)

		Log.Info("request completed",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", time.Since(start)),
			zap.Int("status", lw.data.status),
			zap.Int("size", lw.data.size),
		)
	})
}

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	data *responseData
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	// net/http сам выставляет 200 при Write без WriteHeader, но вызывает
	// WriteHeader у вложенного ResponseWriter, а не у обёртки.
	if w.data.status == 0 {
		w.data.status = http.StatusOK
	}

	size, err := w.ResponseWriter.Write(b)
	w.data.size += size
	return size, err
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.data.status != 0 {
		return
	}

	w.data.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
