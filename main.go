package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/corenzan/parrot/twitter"
)

var (
	logger *log.Logger
	addr   string
	static http.Handler
)

func handler(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

	w.Header().Set("X-Request-Id", requestID)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	switch path.Dir(r.URL.Path) {
	case "/twitter":
		twitter.ServeHTTP(w, r)
	default:
		static.ServeHTTP(w, r)
	}
}

func main() {
	flag.StringVar(&addr, "addr", ":8080", "Server bound address")
	flag.Parse()

	logger = log.New(os.Stdout, "http: ", log.LstdFlags)

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	static = http.FileServer(http.Dir(path.Join(path.Dir(ex), "public")))

	server := &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(handler),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	wait := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logger.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not shutdown the server: %v\n", err)
		}
		close(wait)
	}()

	logger.Println("Server is listening on", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", addr, err)
	}

	<-wait
}
