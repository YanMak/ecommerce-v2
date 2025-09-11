package middleware

import (
	"bufio"
	"net"
	"net/http"
)

type ResponseCapture struct {
	http.ResponseWriter
	Status int
	Bytes  int
}

func NewCapture(w http.ResponseWriter) *ResponseCapture {
	return &ResponseCapture{ResponseWriter: w}
}

func (rc *ResponseCapture) WriteHeader(code int) {
	rc.Status = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *ResponseCapture) Write(b []byte) (int, error) {
	if rc.Status == 0 {
		rc.Status = http.StatusOK
	}
	n, err := rc.ResponseWriter.Write(b)
	rc.Bytes += n
	return n, err
}

// Проксируем доп. интерфейсы, если исходный writer их поддерживает.

func (rc *ResponseCapture) Flush() {
	if f, ok := rc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rc *ResponseCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rc.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (rc *ResponseCapture) Push(target string, opts *http.PushOptions) error {
	if p, ok := rc.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
