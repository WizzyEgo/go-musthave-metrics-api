package middleware

import (
	"bytes"
	"io"
	"net/http"

	"go-musthave-metrics-tpl/internal/hash"
)

// HashSHA256 проверяет подпись тела запроса и подписывает тело ответа.
// Подпись — HMAC-SHA256 от байтов, которые реально переданы в HTTP-теле.
// Пустой ключ отключает проверку и подпись.
//
// Заголовок не обязателен: автотесты и клиенты без ключа ходят на сервер,
// у которого ключ задан. Если HashSHA256 передан и не совпал, данные
// отбрасываются и возвращается 400.
func HashSHA256(key string) func(http.Handler) http.Handler {
	if key == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := readAndRestoreBody(r)
			if err != nil {
				rejectHash(w, key)
				return
			}

			if got := r.Header.Get(hash.Header); got != "" && !hash.Valid(body, key, got) {
				rejectHash(w, key)
				return
			}

			hw := &hashWriter{ResponseWriter: w}
			next.ServeHTTP(hw, r)
			hw.finish(key)
		})
	}
}

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		r.Body = http.NoBody
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	closeErr := r.Body.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func rejectHash(w http.ResponseWriter, key string) {
	hw := &hashWriter{ResponseWriter: w, status: http.StatusBadRequest}
	hw.finish(key)
}

// hashWriter копит ответ, чтобы подписать уже сформированное тело,
// в том числе после gzip.
type hashWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *hashWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
}

func (w *hashWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *hashWriter) finish(key string) {
	w.Header().Set(hash.Header, hash.Sum(w.body.Bytes(), key))
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(status)
	if w.body.Len() == 0 {
		return
	}
	_, _ = w.ResponseWriter.Write(w.body.Bytes())
}
