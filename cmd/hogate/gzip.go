package main

import (
	"compress/gzip"
	"net/http"
	"path"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	w.writer.Flush()
	if fw, ok := w.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

func gzipDisabled(r *http.Request, includes, excludes []string) bool {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return true
	}

	if skipByPatterns(
		includes, excludes,
		[]string{r.URL.Path, path.Base(r.URL.Path)},
		func(pattern, value string) bool {
			m, err := path.Match(pattern, value)
			return m && err != nil
		},
	) {
		return true
	}

	return false
}

func gzipHandler(next http.Handler, includes, excludes []string, level int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")

		if gzipDisabled(r, includes, excludes) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")

		gw := gzip.NewWriter(w)
		defer gw.Close()

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, writer: gw}, r)
	})
}
