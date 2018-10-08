package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/corenzan/parrot/twitter"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

var (
	addr string
)

func main() {
	flag.StringVar(&addr, "addr", ":8080", "Server bound address")
	flag.Parse()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(1 * time.Second))
	r.Use(cors.New(cors.Options{}).Handler)

	r.Get("/twitter/{name}", func(w http.ResponseWriter, r *http.Request) {
		twitter.ServeHTTP(w, r)
	})

	r.Get("/*", http.FileServer(http.Dir("./public")).ServeHTTP)

	http.ListenAndServe(addr, r)
}
