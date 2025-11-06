package common

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func Logf(format string, args ...any) {
	log.Printf("[upstream] "+format, args...)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &respWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		dur := time.Since(start)
		Logf("%s %s -> %d (%s) ua=%q", r.Method, r.URL.String(), rw.status, dur, r.UserAgent())
	})
}

type respWriter struct {
	http.ResponseWriter
	status int
}

func (rw *respWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker by delegating to the underlying ResponseWriter
// when available. It enables WebSocket upgrades to work through the logger wrapper.
func (rw *respWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacker not supported")
}

// Flush passes through to the underlying http.Flusher when supported.
func (rw *respWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprint(w, toJSON(payload))
}

// Text is removed as unused to keep build warnings clean.

func toJSON(v any) string {
	s, ok := v.(string)
	if ok {
		return s
	}
	b, err := jsonMarshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// isolated for easy replacement if needed
var jsonMarshal = func(v any) ([]byte, error) { return json.Marshal(v) }
