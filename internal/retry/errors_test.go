package retry_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"syscall"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"go-musthave-metrics-tpl/internal/retry"
)

func TestConnectionError(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "status", err: errors.New("unexpected status code: 500"), want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "canceled transport", err: &url.Error{Op: "Post", URL: "http://localhost", Err: context.Canceled}, want: false},
		{name: "refused", err: refused, want: true},
		{name: "wrapped refused", err: &url.Error{Op: "Post", URL: "http://localhost", Err: refused}, want: true},
		{name: "eof", err: io.EOF, want: true},
		{name: "reset", err: syscall.ECONNRESET, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retry.Connection(tt.err); got != tt.want {
				t.Fatalf("Connection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresError(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "no rows", err: sql.ErrNoRows, want: false},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "unique violation", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: false},
		{name: "wrapped unique violation", err: fmt.Errorf("insert: %w", &pgconn.PgError{Code: pgerrcode.UniqueViolation}), want: false},
		{name: "connection exception", err: &pgconn.PgError{Code: pgerrcode.ConnectionException}, want: true},
		{name: "connection failure", err: &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, want: true},
		{name: "cannot connect", err: &pgconn.PgError{Code: pgerrcode.SQLClientUnableToEstablishSQLConnection}, want: true},
		{name: "protocol violation", err: &pgconn.PgError{Code: pgerrcode.ProtocolViolation}, want: true},
		{name: "conn closed", err: pgconn.ErrConnClosed, want: true},
		{name: "bad conn", err: driver.ErrBadConn, want: true},
		{name: "timeout", err: context.DeadlineExceeded, want: true},
		{name: "refused", err: refused, want: true},
		{name: "eof", err: io.ErrUnexpectedEOF, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retry.Postgres(tt.err); got != tt.want {
				t.Fatalf("Postgres() = %v, want %v", got, tt.want)
			}
		})
	}
}
