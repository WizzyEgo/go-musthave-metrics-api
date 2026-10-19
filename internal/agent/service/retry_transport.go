package service

import (
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
		attemptReq, cloneErr := cloneRequest(req, attemptCtx, attempt)
		if cloneErr != nil {
			cancel()
			return cloneErr
		}

		var rtErr error
		resp, rtErr = base.RoundTrip(attemptReq)
		if rtErr != nil {
			cancel()
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			resp = nil
			if !retry.Connection(rtErr) || !canRewind(req) {
				return rtErr
			}
			t.log.Warn("retriable connection error", zap.Int("attempt", attempt), zap.Error(rtErr))
			return retry.Retriable(rtErr)
		}

		if resp != nil && resp.Body != nil {
			resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
			return nil
		}
		cancel()
		return nil
	})
	if err != nil {
		return nil, err
	}
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

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}
