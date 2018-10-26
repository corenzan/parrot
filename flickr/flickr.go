package flickr

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	ua       = "Parrot/1.0"
	endpoint = "https://api.flickr.com"
)

// Feed ...
type Feed struct {
	Items []struct {
		Media struct {
			URL    string `xml:"url,attr"`
			Height int    `xml:"height,attr"`
			Width  int    `xml:"width,attr"`
		} `xml:"content"`
		Link string `xml:"link"`
	} `xml:"channel>item"`
}

// HTML ...
func (f Feed) HTML() string {
	var lnk string
	for _, item := range f.Items {
		lnk = lnk + fmt.Sprintf(`<a href="%s"><img src="%s" width="%d" height="%d"></a>`, item.Link, item.Media.URL, item.Media.Width, item.Media.Height)
	}
	return lnk
}

// JSON ...
func (f Feed) JSON() []map[string]string {
	data := make([]map[string]string, 9)
	for i := 0; i < 9; i++ {
		if i >= len(f.Items) {
			break
		}
		data[i] = map[string]string{
			"href":   f.Items[i].Link,
			"src":    f.Items[i].Media.URL,
			"width":  fmt.Sprintf("%d", f.Items[i].Media.Width),
			"height": fmt.Sprintf("%d", f.Items[i].Media.Height),
		}
	}
	return data
}

// Text ...
func (f Feed) String() string {
	var img []string
	for _, item := range f.Items {
		img = append(img, item.Media.URL)
	}
	return strings.Join(img, "\n")
}

// Client ...
type Client struct {
	cache *cache.Cache
	http  *http.Client
}

// New ...
func New() *Client {
	return &Client{
		http: &http.Client{
			Timeout: time.Second * 2,
		},
		cache: cache.New(1*time.Hour, 1*time.Hour),
	}
}

// NewRequest ...
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(endpoint + path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", ua)
	return req, nil
}

// LatestActivity ...
func (c *Client) LatestActivity(username string) (*Feed, error) {
	var feed *Feed
	if value, ok := c.cache.Get(username); ok {
		feed = value.(*Feed)
	} else {
		req, err := c.NewRequest("GET", "/services/feeds/photos_public.gne?format=rss2&id="+username, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(resp.Status)
		}
		err = xml.NewDecoder(resp.Body).Decode(&feed)
		if err != nil {
			return nil, err
		}
		c.cache.Set(username, feed, cache.DefaultExpiration)
	}
	return feed, nil
}

var (
	client *Client
)

func init() {
	client = New()
}

// ServeHTTP ...
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	basename := path.Base(r.URL.Path)
	ext := path.Ext(basename)
	username := strings.TrimSuffix(basename, ext)
	if username == "flickr" {
		http.NotFound(w, r)
		return
	}
	activity, err := client.LatestActivity(username)
	if err != nil {
		panic(err)
	}
	switch ext {
	case ".html", "":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, activity.HTML())
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err := json.NewEncoder(w).Encode(activity.JSON())
		if err != nil {
			panic(err)
		}
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, activity.String())
	default:
		w.WriteHeader(http.StatusNotAcceptable)
	}
}
