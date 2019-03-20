package flickr

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/corenzan/parrot/analytics"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cache "github.com/patrickmn/go-cache"
)

const (
	ua       = "Parrot/1.0"
	endpoint = "https://api.flickr.com"
)

var (
	errNotFound = errors.New("Not Found")
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
		if resp.StatusCode == http.StatusNotFound {
			return nil, errNotFound
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Unexpected Response: %+v", resp)
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

// Route ...
func Route(g *echo.Group) {
	g.Use(middleware.CORS())
	g.Use(analytics.Middleware())

	g.GET("/", func(c echo.Context) error {
		c.NoContent(http.StatusBadRequest)
		return nil
	})

	g.GET("/:basename", func(c echo.Context) error {
		basename := c.Param("basename")
		ext := path.Ext(basename)
		username := strings.TrimSuffix(basename, ext)
		activity, err := client.LatestActivity(username)
		if err != nil {
			if err == errNotFound {
				c.NoContent(http.StatusNotFound)
				return nil
			}
			panic(err)
		}
		switch ext {
		case ".html", "":
			c.HTML(http.StatusOK, activity.HTML())
		case ".json":
			c.JSON(http.StatusOK, activity.JSON())
		case ".txt":
			c.String(http.StatusOK, activity.String())
		default:
			c.NoContent(http.StatusNotAcceptable)
		}
		return nil
	})
}
