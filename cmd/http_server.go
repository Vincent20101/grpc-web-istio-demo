package main

import (
	"crypto/tls"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"

	"golang.org/x/net/http2"
)

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello, HTTP/2.0 h2c!"))
	})

	h2s := &http2.Server{}

	server := &http.Server{
		Addr:         ":12345",
		Handler:      h2cHandler(handler, h2s),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Fatal(server.ListenAndServe())
}

func h2cHandler(h http.Handler, h2s *http2.Server) http.Handler {
	return h2c.NewHandler(h, h2s)
}
