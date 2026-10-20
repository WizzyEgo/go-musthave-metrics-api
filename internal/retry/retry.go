package retry

import (
	"context"
	"errors"
	"time"
)

func backoffDelays() []time.Duration {
	return []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}
}

type retriableError struct {
	err error
}

func (e *retriableError) Error() string {
	if e == nil || e.err == nil {
		return "retriable error"
	}
	return e.err.Error()
}

func (e *retriableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Retriable помечает ошибку как временную: Do повторит вызов.
func Retriable(err error) error {
	if err == nil {
		return nil
	}
	return &retriableError{err: err}
}

// Do вызывает fn и при ошибке Retriable делает ещё три попытки.
// Между попытками выдерживаются паузы 1s, 3s и 5s. Отмена ctx прерывает ожидание.
func Do(ctx context.Context, fn func() error) error {
	return do(ctx, backoffDelays(), fn)
}

func do(ctx context.Context, backoff []time.Duration, fn func() error) error {
	var err error
	for attempt := 0; ; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		err = fn()
		if err == nil {
			return nil
		}

		var retriable *retriableError
		if !errors.As(err, &retriable) || attempt >= len(backoff) {
			if retriable != nil {
				return retriable.err
			}
			return err
		}

		timer := time.NewTimer(backoff[attempt])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
