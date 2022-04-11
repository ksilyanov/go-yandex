package compressor

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(writer, request)
			return
		}

		gz, err := gzip.NewWriterLevel(writer, gzip.BestSpeed)
		if err != nil {
			io.WriteString(writer, err.Error())
			return
		}
		defer gz.Close()

		if strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(request.Body)

			if err != nil {
				io.WriteString(writer, err.Error())
				return
			}
			defer reader.Close()

			body, err := io.ReadAll(reader)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			request.Body = io.NopCloser(bytes.NewReader(body))
		}

		writer.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: writer, Writer: gz}, request)
	})
}
