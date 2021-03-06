package twitter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	endpoint = "https://api.twitter.com"
)

var (
	errNotFound = errors.New("Not Found")
)

// Timeline ...
type Timeline []struct {
	Text     string `json:"text"`
	Entities struct {
		Hashtags []struct {
			Text string `json:"text"`
		} `json:"hashtags"`
		UserMentions []struct {
			ScreenName string `json:"screen_name"`
		} `json:"user_mentions"`
		URLs []struct {
			ExpandedURL string `json:"expanded_url"`
			DisplayURL  string `json:"display_url"`
			URL         string `json:"url"`
		} `json:"urls"`
	} `json:"entities"`
}

// HTML ...
func (t Timeline) HTML() string {
	s := t.String()
	for _, h := range t[0].Entities.Hashtags {
		lnk := `<a href="https://twitter.com/hashtag/` + h.Text + `">#` + h.Text + `</a>`
		s = strings.Replace(s, "#"+h.Text, lnk, 1)
	}
	for _, um := range t[0].Entities.UserMentions {
		lnk := `<a href="https://twitter.com/` + um.ScreenName + `">@` + um.ScreenName + `</a>`
		s = strings.Replace(s, "@"+um.ScreenName, lnk, 1)
	}
	for _, u := range t[0].Entities.URLs {
		lnk := `<a href="` + u.ExpandedURL + `">` + u.DisplayURL + `</a>`
		s = strings.Replace(s, u.URL, lnk, 1)
	}
	return s
}

// JSON ...
func (t Timeline) JSON() map[string]string {
	return map[string]string{
		"status": t.String(),
	}
}

// Text ...
func (t Timeline) String() string {
	return t[0].Text
}

// Client ...
type Client struct {
	credentials string
	cache       *cache.Cache
	http        *http.Client
}

// New ...
func New(key, secret string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: time.Second * 2,
		},
		cache:       cache.New(1*time.Hour, 1*time.Hour),
		credentials: base64.StdEncoding.EncodeToString([]byte(key + ":" + secret)),
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

// AccessToken ...
func (c *Client) AccessToken() (string, error) {
	req, err := c.NewRequest("POST", "/oauth2/token", bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Basic "+c.credentials)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	var token map[string]string
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return "", err
	}
	if token["token_type"] != "bearer" {
		return "", errors.New("Token type is not bearer")
	}
	return token["access_token"], nil
}

// LatestActivity ...
func (c *Client) LatestActivity(username string) (Timeline, error) {
	var timeline Timeline
	if value, ok := c.cache.Get(username); ok {
		timeline = value.(Timeline)
	} else {
		req, err := c.NewRequest("GET", "/1.1/statuses/user_timeline.json?count=1&screen_name="+username, nil)
		if err != nil {
			return nil, err
		}
		token, err := c.AccessToken()
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, errNotFound
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Unexpected Response: %+v", resp)
		}
		err = json.NewDecoder(resp.Body).Decode(&timeline)
		if err != nil {
			return nil, err
		}
		c.cache.Set(username, timeline, cache.DefaultExpiration)
	}
	return timeline, nil
}

var (
	client *Client
)

func init() {
	secret := os.Getenv("TWITTER_SECRET")
	key := os.Getenv("TWITTER_KEY")

	client = New(key, secret)
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
