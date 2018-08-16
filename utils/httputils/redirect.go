package httputils

import (
	"bytes"
	"net/http"
)

type fakeResponseWriter struct {
	headers    http.Header
	statusCode int
	Buffer     *bytes.Buffer
}

func NewFakeWriter() *fakeResponseWriter {
	return &fakeResponseWriter{headers: make(http.Header), Buffer: &bytes.Buffer{}}
}

func (f *fakeResponseWriter) Header() http.Header {
	return f.headers
}

func (f *fakeResponseWriter) Write(data []byte) (int, error) {
	return f.Buffer.Write(data)
}

func (f *fakeResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

func (f *fakeResponseWriter) StatusCode() int {
	return f.statusCode
}

func (f *fakeResponseWriter) Data() []byte {
	return f.Buffer.Bytes()
}

func Redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	if r.Header.Get("X-Requested-With") != "" {
		fw := NewFakeWriter()
		http.Redirect(fw, r, url, code)
		w.Header().Set("X-Location", fw.Header().Get("Location"))
		w.WriteHeader(code)
		if data := fw.Buffer.Bytes(); len(data) > 0 {
			w.Write(data)
		}
	} else {
		http.Redirect(w, r, url, code)
	}
}
