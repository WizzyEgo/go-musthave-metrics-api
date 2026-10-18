package storage

import (
	"database/sql"
	"strings"

	"go.uber.org/zap"
)

// New выбирает хранилище: PostgreSQL, файл, затем память.
func New(log *zap.Logger, dsn string, db *sql.DB, filePath string, synchronous, restore bool) (Storage, *FileStorage, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if strings.TrimSpace(dsn) != "" {
		store, err := NewPostgresStorage(db, log)
		if err != nil {
			return nil, nil, err
		}
		log.Info("using postgres storage")
		return store, nil, nil
	}

	file, err := NewFileStorage(log, filePath, synchronous, restore)
	if filePath == "" {
		log.Info("using memory storage")
	} else {
		log.Info("using file storage", zap.String("path", filePath))
	}
	return file, file, err
}
