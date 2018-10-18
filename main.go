package main

import (
	"flag"
	"net/http"
	"path"
	"time"

	"github.com/corenzan/parrot/instagram"
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

	fs := http.FileServer(http.Dir("./public"))

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 1))
	r.Use(cors.New(cors.Options{}).Handler)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		switch path.Dir(r.URL.Path) {
		case "/instagram":
			instagram.ServeHTTP(w, r)
		case "/twitter":
			twitter.ServeHTTP(w, r)
		default:
			fs.ServeHTTP(w, r)
		}
	})

	http.ListenAndServe(addr, r)
}
