package middleware

import "net/http"

type ResponseWriter struct {
	http.ResponseWriter
	http.Flusher
	http.Hijacker
}

func (wri *ResponseWriter) Write(data []byte) (int, error) {
	n, err := wri.ResponseWriter.Write(data)

	if wri.Flusher != nil {
		wri.Flush()
	}

	return n, err
}

func (wri *ResponseWriter) WriteHeader(statusCode int) {
	wri.ResponseWriter.WriteHeader(statusCode)

	if wri.Flusher != nil {
		wri.Flush()
	}
}
