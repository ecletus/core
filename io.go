package core

import "net/http"

type SentWriterInterface interface {
	http.ResponseWriter
	Sent() bool
}

type sentWriter struct {
	http.ResponseWriter
	sent bool
}

func SentWriter(w http.ResponseWriter) SentWriterInterface {
	if w, ok := w.(SentWriterInterface); ok {
		return w
	}
	return &sentWriter{ResponseWriter: w}
}

func (this *sentWriter) Write(b []byte) (int, error) {
	if len(b) > 0 {
		this.sent = true
	}
	return this.ResponseWriter.Write(b)
}

func (this *sentWriter) WriteHeader(status int) {
	this.sent = true
	this.ResponseWriter.WriteHeader(status)
}

func (this *sentWriter) Sent() bool {
	return this.sent
}
