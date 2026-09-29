package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Gzip распаковывает запрос с Content-Encoding: gzip и сжимает ответ,
// если клиент прислал Accept-Encoding: gzip, а тип контента — application/json или text/html.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decompressRequest(r); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if !acceptsGzip(r) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Add("Vary", "Accept-Encoding")
		cw := &compressWriter{ResponseWriter: w}
		defer cw.Close()
		next.ServeHTTP(cw, r)
	})
}

func decompressRequest(r *http.Request) error {
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip") {
		return nil
	}

	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		return err
	}
	r.Body = &gzipBody{Reader: zr, original: r.Body}
	return nil
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip")
}

func compressible(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "application/json") || strings.Contains(ct, "text/html")
}

type gzipBody struct {
	*gzip.Reader
	original io.Closer
}

func (b *gzipBody) Close() error {
	err := b.Reader.Close()
	if cerr := b.original.Close(); cerr != nil && err == nil {
		err = cerr
	}
	return err
}

type compressWriter struct {
	http.ResponseWriter
	zw      *gzip.Writer
	decided bool
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.decided {
		c.ResponseWriter.WriteHeader(statusCode)
		return
	}
	c.decided = true

	if compressible(c.Header().Get("Content-Type")) {
		c.Header().Set("Content-Encoding", "gzip")
		c.Header().Del("Content-Length")
		zw, err := gzip.NewWriterLevel(c.ResponseWriter, gzip.BestSpeed)
		if err == nil {
			c.zw = zw
		} else {
			c.Header().Del("Content-Encoding")
		}
	}
	c.ResponseWriter.WriteHeader(statusCode)
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.decided {
		c.WriteHeader(http.StatusOK)
	}
	if c.zw != nil {
		return c.zw.Write(p)
	}
	return c.ResponseWriter.Write(p)
}

func (c *compressWriter) Close() error {
	if c.zw == nil {
		return nil
	}
	err := c.zw.Close()
	c.zw = nil
	return err
}
