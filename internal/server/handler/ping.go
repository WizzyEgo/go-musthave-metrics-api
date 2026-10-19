package handler

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func Ping(log *zap.Logger, pinger Pinger) http.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if pinger == nil {
			writeInternalError(log, w, "storage ping failed", nil)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		if err := pinger.Ping(ctx); err != nil {
			writeInternalError(log, w, "storage ping failed", err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
