package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/retry"
)

const attemptTimeout = 5 * time.Second

// retryTransport повторяет запрос при обрыве соединения.
// Таймаут задаётся на каждую попытку, а не на всю цепочку ретраев.
type retryTransport struct {
	base http.RoundTripper
	log  *zap.Logger
}

func newRetryTransport(base http.RoundTripper, log *zap.Logger) *retryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &retryTransport{base: base, log: log}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	attempt := 0
	var resp *http.Response
	err := retry.Do(req.Context(), func() error {
		attempt++
		attemptCtx, cancel := context.WithTimeout(req.Context(), attemptTimeout)
		defer cancel()

		attemptReq, cloneErr := cloneRequest(req, attemptCtx, attempt)
		if cloneErr != nil {
			return cloneErr
		}

		attemptResp, rtErr := roundTripBuffered(base, attemptReq)
		if rtErr != nil {
			if !retry.Connection(rtErr) || !canRewind(req) {
				return rtErr
			}
			t.log.Warn("retriable connection error", zap.Int("attempt", attempt), zap.Error(rtErr))
			return retry.Retriable(rtErr)
		}
		resp = attemptResp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// bodyclose:handled
// roundTripBuffered закрывает исходное тело ответа и отдаёт его копию.
// Повтор запроса не может читать уже закрытое тело, а проверка bodyclose
// не видит Close внутри замыкания ретрая.
func roundTripBuffered(base http.RoundTripper, req *http.Request) (*http.Response, error) {
	resp, err := base.RoundTrip(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(payload))
	return resp, nil
}

func canRewind(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

func cloneRequest(req *http.Request, ctx context.Context, attempt int) (*http.Request, error) {
	clone := req.Clone(ctx)
	if req.Body == nil || req.Body == http.NoBody {
		return clone, nil
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		clone.Body = body
		return clone, nil
	}
	if attempt == 1 {
		clone.Body = req.Body
		return clone, nil
	}
	return nil, errors.New("retry transport: request body is not rewindable")
}
