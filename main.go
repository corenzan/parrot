package main

import (
	"fmt"
	"github.com/corenzan/cockatoo/twitter"
	"github.com/patrickmn/go-cache"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

var (
	routeRegexp = regexp.MustCompile(`^/(\w+)(|\.txt|\.html|.json)$`)
	urlRegexp   = regexp.MustCompile(`https?://\S+`)
)

type Cockatoo struct {
	twitter *twitter.Client
	cache   *cache.Cache
}

func (c *Cockatoo) autoLink(text string) string {
	return urlRegexp.ReplaceAllStringFunc(text, func(src string) string {
		URL, err := url.Parse(src)
		if err != nil {
			return src
		}
		if URL.Scheme == "" {
			URL.Scheme = "http"
		}
		return `<a href="` + URL.String() + `">` + src + `</a>`
	})
}

func (c *Cockatoo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := routeRegexp.FindSubmatch([]byte(r.URL.Path))
	if parts == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	username, format := parts[1], parts[2]
	var status *twitter.Status
	cached, found := c.cache.Get(string(username))
	if found {
		status = cached.(*twitter.Status)
	} else {
		status = c.twitter.LastStatus(string(username))
		c.cache.Set(string(username), status, cache.DefaultExpiration)
	}
	if status == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch string(format) {
	case "", ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, c.autoLink(status.Text))
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, status.Text)
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"status":"`+status.Text+`"}`)
	}
}

func main() {
	key := os.Getenv("TWITTER_KEY")
	secret := os.Getenv("TWITTER_SECRET")
	c := &Cockatoo{
		twitter.New(key, secret),
		cache.New(time.Hour, time.Hour),
	}
	http.Handle("/", c)
	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
