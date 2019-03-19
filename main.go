package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/corenzan/parrot/flickr"
	"github.com/corenzan/parrot/instagram"
	"github.com/corenzan/parrot/twitter"

	raven "github.com/getsentry/raven-go"
)

var (
	addr string
)

func main() {
	raven.SetDSN("")

	flag.StringVar(&addr, "addr", ":8080", "Server bound address")
	flag.Parse()

	fs := http.FileServer(http.Dir("./public"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				err := value.(error)
				http.Error(w, "", http.StatusInternalServerError)

				log.Printf("Panic: %+v\n", err)

				raven.SetHttpContext(raven.NewHttp(r))
				raven.CaptureError(err, nil)
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Header.Get("Origin") != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = fmt.Sprint(time.Now().UnixNano())
		}
		w.Header().Set("X-Request-Id", id)

		route := r.Method + " " + path.Dir(r.URL.Path)
		switch route {
		case "GET /instagram", "POST /instagram":
			instagram.ServeHTTP(w, r)
		case "GET /twitter":
			twitter.ServeHTTP(w, r)
		case "GET /flickr":
			flickr.ServeHTTP(w, r)
		default:
			if r.Method != http.MethodGet {
				http.Error(w, "", http.StatusMethodNotAllowed)
				break
			}
			fs.ServeHTTP(w, r)
		}

		log.Printf("%s %s %s %s %s", id, r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())
	})
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, http.DefaultServeMux))
}
