package main

import (
	"fmt"
	"github.com/corenzan/cockatoo/twitter"
	"log"
	"net/http"
	"os"
	"regexp"
)

var re = regexp.MustCompile(`^/(\w+)(|\.txt|\.html|.json)$`)

type Cockatoo struct {
	twitter *twitter.Client
}

func (c *Cockatoo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := re.FindSubmatch([]byte(r.URL.Path))
	if parts == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	status := c.twitter.LastStatus(string(parts[1]))
	if status == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch string(parts[2]) {
	case "", ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, status.Text)
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, status.Text)
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"status":"`+status.Text+`"}`)
	}
}

func main() {
	t := twitter.New(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"))
	c := &Cockatoo{t}
	http.Handle("/", c)
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
