package compress

import (
	"bytes"
	"compress/gzip"
	"io"
)

// NewWriter возвращает gzip.Writer с тем же уровнем сжатия, что использует агент и сервер.
func NewWriter(w io.Writer) (*gzip.Writer, error) {
	return gzip.NewWriterLevel(w, gzip.BestSpeed)
}

// NewReader распаковывает gzip-поток.
func NewReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}

// Encode сжимает data в gzip.
func Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err = zw.Write(data); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
