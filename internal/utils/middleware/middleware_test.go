package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareResponseWriterFlush(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
		mWri := &ResponseWriter{ResponseWriter: wri}
		if f, ok := wri.(http.Flusher); ok {
			mWri.Flusher = f

			mWri.WriteHeader(http.StatusTeapot)
			_, _ = mWri.Write([]byte("I'm a teapot"))
		} else {
			mWri.WriteHeader(http.StatusInternalServerError)
			_, _ = mWri.Write([]byte("I'm NOT a teapot"))
		}
	}))
	defer ts.Close()

	res, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusTeapot)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "I'm a teapot" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "I'm a teapot")
	}
}

func TestMiddlewareResponseWriterHijack(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
		if hj, ok := wri.(http.Hijacker); ok {
			down, downRw, err := hj.Hijack()
			if err != nil {
				return
			}
			defer func() {
				_ = down.Close()
			}()

			_, _ = downRw.Write([]byte("HTTP/1.1 418\r\n\r\nI'm a teapot"))
			_ = downRw.Flush()
		} else {
			wri.WriteHeader(http.StatusInternalServerError)
			_, _ = wri.Write([]byte("I'm NOT a teapot"))
		}
	}))
	defer ts.Close()

	res, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusTeapot)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "I'm a teapot" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "I'm a teapot")
	}
}
