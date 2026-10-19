package retry

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"syscall"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// Connection — временный обрыв связи с сервером: соединение не установилось
// или оборвалось до ответа. Отмена операции не повторяется.
func Connection(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return isBrokenPipe(err)
}

// Postgres — ошибка, после которой запрос к PostgreSQL можно повторить.
// Class 08 (Connection Exception) определяется через pgerrcode.
// UniqueViolation и прочие ошибки данных не повторяются: это не сбой транспорта.
// Сетевой обрыв до ответа сервера тоже повторяется: SQLSTATE в этом случае нет.
func Postgres(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, sql.ErrNoRows) {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return false
		}
		return pgerrcode.IsConnectionException(pgErr.Code)
	}

	if errors.Is(err, pgconn.ErrConnClosed) || errors.Is(err, driver.ErrBadConn) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return Connection(err)
}

func isBrokenPipe(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ETIMEDOUT)
}
